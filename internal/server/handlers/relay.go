package handlers

import (
	"net/http"

	"github.com/looplj/axonhub/llm"
	"github.com/shengmingboai/octopus/internal/relay"
	"github.com/shengmingboai/octopus/internal/server/middleware"
	"github.com/shengmingboai/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth()).
		AddRoute(
			router.NewRoute("/chat/completions", http.MethodPost).
				Handle(relay.Forward(llm.APIFormatOpenAIChatCompletion)),
		).
		AddRoute(
			router.NewRoute("/responses", http.MethodPost).
				Handle(relay.Forward(llm.APIFormatOpenAIResponse)),
		).
		AddRoute(
			router.NewRoute("/messages", http.MethodPost).
				Handle(relay.Forward(llm.APIFormatAnthropicMessage)),
		)
}
