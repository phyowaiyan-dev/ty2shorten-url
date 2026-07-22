package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phyowaiyan-dev/ty2shorten-url/internal/services"
)

// Analytics tracks eligible public page views and redirects after handlers complete.
func Analytics(analytics *services.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		if analytics != nil {
			analytics.Prepare(c)
		}
		c.Next()
		if analytics == nil || c.IsAborted() {
			return
		}
		requestID, _ := c.Get("request_id")
		options := services.AnalyticsTrackOptions{
			RequestID:   stringifyMiddlewareValue(requestID),
			StatusCode:  c.Writer.Status(),
			Duration:    time.Since(startedAt),
			Destination: c.Writer.Header().Get("Location"),
		}
		if value, ok := c.Get("analytics_route_type"); ok {
			options.RouteType = stringifyMiddlewareValue(value)
		}
		if value, ok := c.Get("analytics_destination_platform"); ok {
			options.DestinationPlatform = stringifyMiddlewareValue(value)
		}
		if value, ok := c.Get("analytics_short_link_slug"); ok {
			options.ShortLinkSlug = stringifyMiddlewareValue(value)
		}
		if value, ok := c.Get("analytics_page_title"); ok {
			options.PageTitle = stringifyMiddlewareValue(value)
		}
		if value, ok := c.Get("analytics_short_link_id"); ok {
			switch id := value.(type) {
			case uint:
				options.ShortLinkID = &id
			case *uint:
				options.ShortLinkID = id
			}
		}
		analytics.Track(c, options)
	}
}

func stringifyMiddlewareValue(value any) string {
	if stringValue, ok := value.(string); ok {
		return stringValue
	}
	return ""
}
