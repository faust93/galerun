// web ui
package main

import (
    "log"
    "strconv"
    "strings"
    "github.com/gin-gonic/gin"
    "net/http"
    "github.com/gin-contrib/sessions"
    _ "github.com/gin-contrib/sessions/cookie"
    "golang.org/x/crypto/bcrypt"
)

type imgItem struct {
    Type string
    Name string
    Path string
    Info string
    Size string
}

func webVideos(c *gin.Context) {
    flist := getVideoList(conf.BasePath);
    log.Println(flist)

    user := User{}
    session := sessions.Default(c)
    userid := session.Get("user")
    if err := db.Read("users", userid.(string), &user); err != nil {
        log.Println("Error", err)
        return
    }

    c.HTML(http.StatusOK, "videos.html",
        gin.H{
            "title": "Videos",
            "user": userid.(string)[0:2],
            "payload": flist,
            "thumbS": user.Prefs.ThumbS,
            "scaleF": user.Prefs.ScaleF,
            })
}

func webIndex(c *gin.Context) {
        c.Redirect(302,"/web/images")
}

func webImages(c *gin.Context) {

        var files = make([]imgItem,0)
        var dirs = make([]imgItem,0)
        var img imgItem

        sort_order := c.DefaultQuery("s","dsc")
        path := c.Query("d")
        if validatePath(path) {
            log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, path)
            return
        }
        path_elems := strings.Split(path,"/")
        flist := getDirList(path)

        for _, item := range flist {
            if item.Type == "f" {
                img = imgItem{"f",item.Name,path,item.Name,ByteConvert(item.Size)}
                files = append(files,img)
            } else if item.Type == "d" {
                img = imgItem{"d",item.Name,path,item.Name,strconv.FormatInt(item.Size, 10)}
                dirs = append(dirs,img)
            }
        }

        if sort_order == "dsc" {
            var tmpf = make([]imgItem,0)
            for i := len(files)-1; i >= 0; i-- {
                tmpf = append(tmpf,files[i])
            }
            files = tmpf
        }
        // Kinda sorting, dirs going first.
        for _, v := range files {
            dirs = append(dirs, v)
        }

        user := User{}
        session := sessions.Default(c)
        userid := session.Get("user")
        if err := db.Read("users", userid.(string), &user); err != nil {
            log.Println("Error", err)
            return
        }

        c.HTML(http.StatusOK, "images.html",
        gin.H{
            "title": "Images",
            "user": userid.(string)[0:2],
            "payload": dirs,
            "path": path_elems,
            "thumbS": user.Prefs.ThumbS,
            "scaleF": user.Prefs.ScaleF,
            "sort": sort_order,
            })
}

func settingsPage(c *gin.Context) {
//        ref := c.GetHeader("Referer")
        user := User{}
        session := sessions.Default(c)
        userid := session.Get("user")
        if err := db.Read("users", userid.(string), &user); err != nil {
            log.Println("Error", err)
        }

        if c.Request.Method == "GET" {
        c.HTML(http.StatusOK, "settings.html",
        gin.H{
            "title": "Settings",
            "user": userid.(string)[0:2],
            "thumbs": conf.ThumbSize,
            "thumbSel": user.Prefs.ThumbS,
            "scales": conf.ScaleFactor,
            "scaleSel": user.Prefs.ScaleF,
            })
        } else if c.Request.Method == "POST" {
            c.Request.ParseForm()
            thumb_s := c.PostForm("thumb_size")
            scale_f := c.PostForm("scale_factor")
            user.Prefs.ThumbS, _ = strconv.Atoi(thumb_s)
            user.Prefs.ScaleF, _ = strconv.ParseFloat(scale_f, 64)
            session := sessions.Default(c)
            userid := session.Get("user")
            if err := db.Write("users", userid.(string), user); err != nil {
                log.Println("Error", err)
            }
            c.Redirect(302,"/web/settings")
        }
}

func loginPage(c *gin.Context) {
        if c.Request.Method == "GET" {
            c.HTML(http.StatusOK, "login.html",gin.H{"title": "Login","status": ""})
        } else if c.Request.Method == "POST" {
            c.Request.ParseForm()
            userid := c.PostForm("user")
            password := c.PostForm("password")

            if strings.Trim(userid, " ") == "" || strings.Trim(password, " ") == "" {
                log.Println("Got empty parameters")
                c.HTML(http.StatusBadRequest, "login.html",gin.H{"title": "Login","status": "Parameters can't be empty"})
                return
            }

            user := User{}
            log.Println("userid ", userid)
            if err := db.Read("users", userid , &user); err != nil {
                log.Println("No such user. Error", err)
                c.HTML(http.StatusBadRequest, "login.html",gin.H{"title": "Login","status": "User/Password is invalid"})
                return
            }
            if bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(password)) == nil && user.Name == userid {
                session := sessions.Default(c)
                session.Set("user", userid)
                err := session.Save()
                if err != nil {
                    log.Println("Failed to generate session token")
                    c.HTML(http.StatusBadRequest, "login.html",gin.H{"title": "Login","status": "Something went wrong"})
                    return
                } else {
                    log.Println("Successfully authenticated user")
                    c.Redirect(302,"/")
                }
            } else {
                    log.Println("Invalid password")
                    c.HTML(http.StatusBadRequest, "login.html",gin.H{"title": "Login","status": "User/Password is invalid"})
                    return
            }

        }
}

func logout(c *gin.Context) {
    session := sessions.Default(c)
    user := session.Get("user")
    if user == nil {
        c.String(http.StatusBadRequest, "Invalid session token")
        return
    } else {
        session.Delete("user")
        session.Clear()
        session.Save()
        c.Redirect(302,"/")
    }
}

func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        user := session.Get("user")
        if user == nil {
            log.Printf("%s Unauthorized access attempt", c.Request.RemoteAddr)
            c.Abort()
            c.Redirect(302,"/login")
        } else {
            c.Next()
        }
    }
}