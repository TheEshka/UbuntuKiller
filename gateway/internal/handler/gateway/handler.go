package gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/internal/handler/common"
	"github.com/Shopify/sarama"
	"github.com/go-chi/jwtauth/v5"
	"io/ioutil"
	"net/http"
)

type Services struct {
	GatewayLogin		string `default:"gateway" split_words:"true"`
	GatewayPassword		string `required:"true" split_words:"true"`

	Scheme				string `default:"http" split_words:"true"`
	SessionService   	string `default:"localhost:9111" split_words:"true"`
	LibraryService   	string `default:"localhost:9114" split_words:"true"`
}

type Handler struct {
	services Services
	client *http.Client
	interServiceTokens map[string]string
	producer *sarama.AsyncProducer
	//(*h.producer).Input() <- &sarama.ProducerMessage{
	//Topic: "njeb2phw-books-genres",
	//Value: sarama.StringEncoder(fmt.Sprintf("{\"genre\": \"%s\"}", )),
}

func New(services Services,  producer *sarama.AsyncProducer) *Handler {
	return &Handler{
		services: services,
		interServiceTokens: make(map[string]string, 0),
		client: &http.Client{},
		producer: producer,
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (h *Handler) AuthChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := fmt.Sprintf("%s://%s/verify", h.services.Scheme, h.services.SessionService)
		req, _ := http.NewRequest("POST", path, nil)
		req.Header.Add("Authorization", r.Header.Get("Authorization"))

		resp, err := h.client.Do(req)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, errors.New(fmt.Sprintf("Internal error requesting %s", r.URL.String())))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			common.RespondError(nil, w, http.StatusUnauthorized, errors.New("не имеете нужных прав"))
			return
		}
		// Token is authenticated, pass it through
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) AdminChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := jwtauth.TokenFromHeader(r)
		if token == "" {
			common.RespondError(nil, w, http.StatusUnauthorized, errors.New("no token"))
			return
		}

		path := fmt.Sprintf("%s://%s/isUserAdmin/%s", h.services.Scheme, h.services.SessionService, token)
		req, _ := http.NewRequest("POST", path, nil)

		err := h.interServiceAuth(h.services.SessionService, req)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, err)
			return
		}

		resp, err := h.client.Do(req)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, errors.New(fmt.Sprintf("Internal error requesting %s", r.URL.String())))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			common.RespondError(nil, w, http.StatusForbidden, err)
			return
		}
		// Token is authenticated, pass it through
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) interServiceAuth(host string, r *http.Request) error {
	if val, ok := h.interServiceTokens[host]; ok {
		r.Header.Add("Authorization","BEARER: " + val)
		return nil
	}
	path := fmt.Sprintf("%s://%s/auth", h.services.Scheme, host)

	req, _ := http.NewRequest("POST", path, nil)

	auth := h.services.GatewayLogin + ":" + h.services.GatewayPassword
	req.Header.Add("Authorization","Basic " + base64.StdEncoding.EncodeToString([]byte(auth)))

	res, err := h.client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("Internal auth error %s", path))
	}
	defer res.Body.Close()

	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Internal error reading from %s", path))
	}

	var jsonMap map[string]string
	err = json.Unmarshal(payload, &jsonMap)
	if err != nil {
		return errors.New(fmt.Sprintf("Internal error reading from %s", path))
	}
	r.Header.Add("Authorization","BEARER: " + jsonMap["token"])
	return nil
}

func (h *Handler) ProxyHandler(host string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = host
		r.URL.Scheme = h.services.Scheme
		r.RequestURI = ""

		err := h.interServiceAuth(host, r)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, err)
			return
		}

		resp, err := h.client.Do(r)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, errors.New(fmt.Sprintf("Internal error requesting %s", r.URL.String())))
			return
		}

		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)

		payload, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			common.RespondError(nil, w, http.StatusInternalServerError, errors.New(fmt.Sprintf("Internal error reading from %s", r.URL.String())))
			return
		}

		common.RespondJSON(nil, w, resp.StatusCode, string(payload))
	}
}

func (h *Handler) GetUserRating(w http.ResponseWriter, r *http.Request) {
	//ctx := r.Context()
	//
	//userUid := chi.URLParam(r, "userUid")
	//
	//q := "SELECT rate FROM ratings WHERE user_uid = $1"
	//var b int
	//if err := h.db.GetContext(ctx, &b, q, userUid); err == pgx.ErrNoRows {
	//	common.Respond(ctx, w, http.StatusNotFound)
	//	return
	//} else if err != nil {
	//	common.RespondError(ctx, w, http.StatusInternalServerError, err)
	//	return
	//}
	//
	//common.RespondJSON(ctx, w, http.StatusOK, b)
}
