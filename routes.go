package main
import(
    "github.com/gin-gonic/gin"
    "net/http"
)

func Options(c *gin.Context) {
    if c.Request.Method != "OPTIONS" {
       c.Header("Access-Control-Allow-Origin", "file://")
       c.Header("Access-Control-Allow-Credentials", "true")
       c.Next()
    } else {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
        c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
//        c.Header("Content-Type", "application/json")
        c.AbortWithStatus(http.StatusOK)
    }
}

func initRoutes() {

    r.Use(Options)

    r.GET("/", webIndex)

    r.GET("/login", loginPage)
    r.POST("/login", loginPage)
    r.GET("/logout", logout)

    web := r.Group("/web")
    web.Use(AuthRequired())
    {
        web.GET("/", webIndex)
        web.GET("/images", webImages)
        web.GET("/videos", webVideos)
        web.GET("/settings", settingsPage)
        web.POST("/settings", settingsPage)
    }

    r.POST("/apiAuth", apiAuth)
    api := r.Group("/api")
    api.Use(AuthRequired())
    {
        api.GET("/img", getImage)
        api.GET("/vid", getVideo)
        api.GET("/list", listImages)
        api.GET("/thumb", getThumbnail)
        api.GET("/exif", getExif)
        api.GET("/info", getImgInfo)
        api.GET("/del", deleteFile)
        api.POST("/del", deleteFile)
        api.POST("/move", moveFiles)
        api.DELETE("/del", deleteFile)
        api.GET("/cmd", dispatchCmd)
        api.POST("/upload", uploadFile)
    }
}

