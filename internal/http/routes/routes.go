package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/http/handlers"
	subsysteroutes "github.com/routerarchitects/ow-common-mods/fiber/system-routes"
)

type PublicDeps struct {
	AuthHandler         fiber.Handler
	Subsystem           subsysteroutes.Config
	PolicyHandler       *handlers.PolicyHandler
	DeviceHandler       *handlers.DeviceHandler
	DeviceEventsHandler *handlers.DeviceEventsHandler
}

type PrivateDeps struct {
	AuthHandler fiber.Handler
	Subsystem   subsysteroutes.Config
}

// RegisterPublic configures the public HTTP router paths.
func RegisterPublic(app *fiber.App, deps PublicDeps) {
	registerLivenessRoute(app)

	// Register production OpenWiFi WebSocket routes (bypasses upfront HTTP Auth, enforces in-band token handshake)
	if deps.DeviceEventsHandler != nil {
		app.Get("/api/v1/ws", deps.DeviceEventsHandler.HandleWebSocket)
		app.Get("/ws", deps.DeviceEventsHandler.HandleWebSocket)
	}

	// Propagate ?token= query parameter to Authorization header for clients that cannot
	// pass custom headers (e.g. browser EventSource for Server-Sent Events).
	app.Use(func(c fiber.Ctx) error {
		if c.Get("Authorization") == "" {
			if token := c.Query("token"); token != "" {
				c.Request().Header.Set("Authorization", "Bearer "+token)
			}
		}
		return c.Next()
	})

	// Create authenticated route group
	group := app.Group("", deps.AuthHandler)

	// Register system diagnostics routes
	subsysteroutes.RegisterRoutes(deps.Subsystem, group)

	// Register policy overview route
	if deps.PolicyHandler != nil {
		group.Get("/api/v1/policy/:id/overview", deps.PolicyHandler.GetOverview)
	}

	// Register real-time device event routes (register before /devices/:serial parameter wildcard)
	if deps.DeviceEventsHandler != nil {
		group.Get("/api/v1/devices/events", deps.DeviceEventsHandler.StreamEvents)
		group.Post("/api/v1/devices/events/subscriptions", deps.DeviceEventsHandler.UpdateSubscriptions)
	}

	// Register device routes
	if deps.DeviceHandler != nil {
		// Blacklist routes (register before /api/v1/devices/:serial wildcard)
		group.Get("/api/v1/devices/blacklist", deps.DeviceHandler.GetBlacklist)
		group.Post("/api/v1/devices/blacklist", deps.DeviceHandler.AddBlacklistDevice)
		group.Delete("/api/v1/devices/blacklist/:serial", deps.DeviceHandler.DeleteBlacklistDevice)
		group.Post("/api/v1/devices/:serial/blacklist", deps.DeviceHandler.AddBlacklistDevice)
		group.Delete("/api/v1/devices/:serial/blacklist", deps.DeviceHandler.DeleteBlacklistDevice)

		group.Get("/api/v1/devices", deps.DeviceHandler.GetDevices)
		group.Get("/api/v1/devices/:serial/commands", deps.DeviceHandler.GetDeviceCommands)
		group.Delete("/api/v1/devices/:serial/commands/:uuid", deps.DeviceHandler.DeleteDeviceCommand)
		group.Get("/api/v1/devices/:serial/healthchecks", deps.DeviceHandler.GetDeviceHealthChecks)
		group.Get("/api/v1/devices/:serial/logs", deps.DeviceHandler.GetDeviceLogs)
		group.Get("/api/v1/devices/:serial/statistics", deps.DeviceHandler.GetDeviceStatistics)
		group.Get("/api/v1/devices/:serial/status", deps.DeviceHandler.GetDeviceStatus)
		group.Get("/api/v1/devices/:serial/capabilities", deps.DeviceHandler.GetDeviceCapabilities)
		group.Get("/api/v1/devices/:serial/rtty", deps.DeviceHandler.GetDeviceRtty)
		group.Get("/api/v1/devices/:serial", deps.DeviceHandler.GetDevice)
		group.Put("/api/v1/devices/:serial", deps.DeviceHandler.UpdateDevice)
		group.Delete("/api/v1/devices/:serial", deps.DeviceHandler.DeleteDevice)
		group.Post("/api/v1/devices/:serial/:command", deps.DeviceHandler.ExecuteDeviceCommand)
	}
}

// RegisterPrivate configures the private/internal HTTP router paths.
func RegisterPrivate(app *fiber.App, deps PrivateDeps) {
	registerLivenessRoute(app)

	// Create authenticated route group
	group := app.Group("", deps.AuthHandler)

	// Register system diagnostics routes
	subsysteroutes.RegisterRoutes(deps.Subsystem, group)
}

func registerLivenessRoute(app *fiber.App) {
	app.Get("/livez", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
}
