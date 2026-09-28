package routes

import (
	"api-kms/enums"
	"api-kms/utils"

	"github.com/gin-gonic/gin"
)

// The two cache endpoints live here rather than in an app/ package because
// there is no domain behind them — no repository, no service, no state of
// their own beyond the map in utils. They are the browser twin of the SIGHUP
// handler in main.go, for an admin who has no shell on the server.

// listCache godoc
// @Summary      List what is cached
// @Description  Every live cache key with how many items it holds and how long it has left. Expired keys are left out — they are already dead to the cache.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  utils.HTTPResponse{payload=object{data=[]utils.CacheEntry}}
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/cache [get]
func listCache(c *gin.Context) error {
	utils.ResponseJSON(c, enums.SUCCESS, "Cache retrieved", utils.CacheEntries())
	return nil
}

// clearCache godoc
// @Summary      Clear the cache
// @Description  Without `key`, drops everything — the same reach as sending the process SIGHUP. With `key`, drops just that entry. Writes made through this app already invalidate what they change, so this is for the one case that cannot — another app writing a shared table (users, roles) directly.
// @Tags         management
// @Produce      json
// @Security     BearerAuth
// @Param        key  query     string  false  "Cache key to drop, e.g. lookup:areas. Omit to clear everything."
// @Success      200  {object}  utils.HTTPResponse
// @Failure      401  {object}  utils.HTTPError
// @Failure      403  {object}  utils.HTTPError
// @Router       /management/cache [delete]
func clearCache(c *gin.Context) error {
	// An empty key is the whole cache, which is what Forget's prefix matching
	// already means — so no branch, and no separate "clear all" endpoint.
	key := c.Query("key")
	utils.Forget(key)

	message := "Cache cleared"
	if key != "" {
		message = "Cache cleared: " + key
	}

	utils.ResponseJSON(c, enums.SUCCESS, message, nil)
	return nil
}
