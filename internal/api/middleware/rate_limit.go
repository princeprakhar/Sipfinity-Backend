package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"fmt"
)

func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	rate := limiter.Rate{
		Period: 1,
		Limit:  int64(cfg.RateLimitRPS),
	}
	
	store := memory.NewStore()
	instance := limiter.New(store, rate, limiter.WithTrustForwardHeader(true))
	
	return mgin.NewMiddleware(instance, mgin.WithKeyGetter(func(c *gin.Context) string {
		return fmt.Sprintf("%s:%s", c.ClientIP(), c.Request.URL.Path)
	}))
}