package handlers

import (
	"errors"
	"html"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/view"
	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// AgentHandler handles agent-related HTTP requests.
type AgentHandler struct {
	agentAuthSvc ports.AgentAuthService
	agentRepo    ports.AgentRepository
	templates    *view.Templates
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(
	agentAuthSvc ports.AgentAuthService,
	agentRepo ports.AgentRepository,
	templates *view.Templates,
) *AgentHandler {
	return &AgentHandler{
		agentAuthSvc: agentAuthSvc,
		agentRepo:    agentRepo,
		templates:    templates,
	}
}

// Create handles agent creation and returns the API key (shown once).
func (h *AgentHandler) Create(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	name := c.FormValue("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "agent name is required"})
	}
	if len(name) > 255 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "agent name must be 255 characters or less"})
	}

	agent, apiKey, err := h.agentAuthSvc.CreateAgent(c.Request().Context(), userID.String(), name)
	if err != nil {
		if errors.Is(err, domain.ErrAgentLimitReached) {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.HTML(http.StatusForbidden, `
					<div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4 mb-4">
						<p class="text-yellow-400 font-medium">Agent limit reached</p>
						<p class="text-gray-400 text-sm mt-1">Upgrade your plan to create more agents.</p>
					</div>`)
			}
			return c.JSON(http.StatusForbidden, map[string]string{"error": "agent limit reached for current plan"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create agent"})
	}

	// If HTMX request, return HTML with the API key displayed
	if c.Request().Header.Get("HX-Request") == "true" {
		escapedName := html.EscapeString(agent.Name)
		escapedKey := html.EscapeString(apiKey)
		return c.HTML(http.StatusOK, `
			<div class="bg-emerald-500/10 border border-emerald-500/20 rounded-md p-3 space-y-2">
				<div class="flex items-center space-x-2">
					<i data-lucide="check-circle" class="w-4 h-4 text-emerald-400"></i>
					<span class="text-xs font-medium text-emerald-400">Agent "`+escapedName+`" created!</span>
				</div>
				<p class="text-[11px] text-muted-foreground">Save this API key now — it won't be shown again:</p>
				<div class="flex items-center space-x-2">
					<code id="agent-key-text" class="flex-1 text-xs font-mono bg-background px-3 py-2 rounded border border-border text-foreground select-all break-all">`+escapedKey+`</code>
					<button onclick="navigator.clipboard.writeText(document.getElementById('agent-key-text').textContent); this.innerHTML='<i data-lucide=&quot;check&quot; class=&quot;w-3.5 h-3.5&quot;></i>'; lucide.createIcons({nodes: [this]})"
						class="shrink-0 px-2 py-2 bg-muted rounded-md text-muted-foreground hover:text-foreground transition-colors">
						<i data-lucide="copy" class="w-3.5 h-3.5"></i>
					</button>
				</div>
				<p class="text-[11px] text-muted-foreground/60">Then run: <code class="text-muted-foreground break-all">watchdog-agent -hub "wss://usewatchdog.dev/ws/agent" -api-key "YOUR_KEY"</code></p>
			</div>
			<script>lucide.createIcons();</script>`)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"agent": map[string]string{
			"id":   agent.ID.String(),
			"name": agent.Name,
		},
		"api_key": apiKey,
	})
}

// Delete handles agent deletion.
func (h *AgentHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent ID"})
	}

	// Verify agent belongs to user
	agent, err := h.agentRepo.GetByID(ctx, id)
	if err != nil || agent == nil || agent.UserID != userID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
	}

	if err := h.agentRepo.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete agent"})
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/dashboard")
}
