package main

import(
    "os"
    "fmt"
    "log"
    "regexp"
    "strings"
    "strconv"
    "io/ioutil"
    "path/filepath"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
    _ "github.com/gin-contrib/sessions/cookie"
    "golang.org/x/crypto/bcrypt"
    fbgo "github.com/facebookgo/symwalk"
)

type RESPONSE struct {
    Filename string `json:"filename"`
    Message string `json:"message"`
    Error ERROR `json:"error"`
}
type ERROR struct {
    HasError     bool   `json:"has_error"`
    ErrorNumber  int    `json:"error_number"`
    ErrorMessage string `json:"error_message"`
}


func getImgInfo(c *gin.Context) {
    resp := RESPONSE{}
    var out strings.Builder

    filename := c.Query("f")
    if validatePath(filename) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, filename)
        return
    }

    f, err := os.Stat(conf.BasePath + filename)
    if err != nil {
        log.Println("Unable to open " + filename)
        return
    }

    if f.IsDir() {
        var size int64
        n, _ := ioutil.ReadDir(conf.BasePath + filename)
        items := len(n)
        err := fbgo.Walk(conf.BasePath + filename, func(_ string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if !info.IsDir() {
                size += info.Size()
            }
            return nil
        })
        if err != nil {
            log.Printf("Error getting %s info: %s", filename, err)
                resp.Error.HasError = true
                resp.Error.ErrorNumber = 3
                resp.Error.ErrorMessage = "Unable to get directory info"
                c.JSON(
                    http.StatusBadRequest,
                    gin.H{
                    "data": resp,
                    })
            return
        }

        s := fmt.Sprintf("<b>%s</b><br><br>", f.Name())
        out.WriteString(s)
        s = fmt.Sprintf("Size is %s<br>", ByteConvert(size))
        out.WriteString(s)
        s = fmt.Sprintf("%s item(s) inside<br>", strconv.Itoa(items))
        out.WriteString(s)

        resp.Filename = filename
        resp.Message = out.String()
        c.JSON(http.StatusOK, gin.H{
            "data": resp,
        })
    } else {
        exif, _ := fetchExif(filename)
        s := fmt.Sprintf("<b>%s</b><br><br>", f.Name())
        out.WriteString(s)
        s = fmt.Sprintf("Size is %s<br>", ByteConvert(f.Size()))
        out.WriteString(s)

        if(exif != "" ){
            out.WriteString("<br>")
            out.WriteString(exif)
        } else {
            s = fmt.Sprintf("Date is %s", f.ModTime().Format("2006-01-02"))
            out.WriteString(s)
        }

        resp.Filename = filename
        resp.Message = out.String()
        c.JSON(http.StatusOK, gin.H{
            "data": resp,
        })
    }
}

func getExif(c *gin.Context) {
    resp := RESPONSE{}

    filename := c.Query("f")
    if validatePath(filename) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, filename)
        return
    }

    exif, err := fetchExif(filename)
    if(err == nil) {
        resp.Filename = filename
        resp.Message = exif
        c.JSON(http.StatusOK, gin.H{
            "data": resp,
        })
    }
}

// commands dispatcher
func dispatchCmd(c *gin.Context) {
    resp := RESPONSE{}

    cmd := c.Query("c")
    arg := c.Query("p")

    switch cmd {
        case "dir_create":
            if validatePath(arg) {
                log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, arg)
                return
            }
            if _, err := os.Stat(conf.BasePath + "/" + arg); os.IsNotExist(err) {
                err = os.Mkdir(conf.BasePath + "/" + arg, 0755)
                if err != nil {
                    log.Printf("Error: %s", err)
                    resp.Error.HasError = true
                    resp.Error.ErrorNumber = 2
                    resp.Error.ErrorMessage = "Unable to create directory"
                    c.JSON(
                        http.StatusBadRequest,
                        gin.H{
                        "data": resp,
                        })
                    return
                }
                resp.Filename = arg
                resp.Message = "Directory created succesfully"
                c.JSON(http.StatusOK, gin.H{
                "data": resp,
                })
            }
    }
}

type List struct {
    Files []string `binding:"required"`
    }

func deleteFile(c *gin.Context) {
    resp := RESPONSE{}

    if c.Request.Method == "POST" {
        data := new(List)
        err := c.Bind(data)
        if err != nil {
            log.Printf("Error: %s", err)
            c.String(http.StatusBadRequest, "Bad Request")
            return
        }
        for _, f := range data.Files {
            if validatePath(f) {
                log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, f)
                c.String(http.StatusBadRequest, "Bad Request")
                return
            }

            err := os.Remove(conf.BasePath + f)
            if err != nil {
                log.Printf("Unable to delete: %s", err)
                resp.Error.HasError = true
            }
        }
        if resp.Error.HasError {
            resp.Error.ErrorNumber = 5
            resp.Error.ErrorMessage = "Cannot remove files"
            c.JSON(http.StatusBadRequest,gin.H{
            "data": resp,
            })
            return
        } else {
            resp.Message = "Files are removed"
            c.JSON(http.StatusOK, gin.H{
            "data": resp,
            })
        }
    } else {
    filename := c.Query("f")
    if validatePath(filename) {
        log.Println("%s Path traversal attempt: %s", c.Request.RemoteAddr, filename)
        return
    }

    var err error
    info, _ := os.Stat(conf.BasePath + filename)
    if info.IsDir() {
        err = os.RemoveAll(conf.BasePath + filename)
    } else {
        err = os.Remove(conf.BasePath + filename)
    }
    if err != nil {
        log.Printf("Error: %s", err)
        resp.Error.HasError = true
        resp.Error.ErrorNumber = 1
        resp.Error.ErrorMessage = "Cannot remove file."
        c.JSON(
            http.StatusBadRequest,
            gin.H{
                "data": resp,
            },
        )
        return
    }

    resp.Filename = filename
    resp.Message = "Deleted"
    c.JSON(http.StatusOK, gin.H{
        "data": resp,
    })
    }
}

type mvList struct {
    Dst   string `binding:"required"`
    Files []string `binding:"required"`
    }

func moveFiles(c *gin.Context) {
    resp := RESPONSE{}

    data := new(mvList)
    err := c.Bind(data)
    if err != nil {
        log.Printf("Error: %s", err)
        c.String(http.StatusBadRequest, "Bad Request")
        return
    }

    if validatePath(data.Dst) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, data.Dst)
        c.String(http.StatusBadRequest, "Bad Request")
        return
    }

    for _, f := range data.Files {
        if validatePath(f) {
            log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, f)
            c.String(http.StatusBadRequest, "Bad Request")
            return
        }
        filename := filepath.Base(f)
        dir := filepath.Dir(f)
        if dir != data.Dst {
            err := os.Rename(conf.BasePath + f, conf.BasePath + data.Dst + "/" + filename)
            if err != nil {
                log.Printf("Unable to move: %s", err)
                resp.Error.HasError = true
            }
        } else {
                log.Printf("Skip moving to the same destination: %s -> %s", f, data.Dst)
        }
    }

    if resp.Error.HasError {
        resp.Error.ErrorNumber = 6
        resp.Error.ErrorMessage = "Cannot move some files"
        c.JSON(http.StatusBadRequest,gin.H{
        "data": resp,
        })
        return
    } else {
        resp.Message = "Files have been moved succesfully"
        c.JSON(http.StatusOK, gin.H{
        "data": resp,
        })
    }
}

func listImages(c *gin.Context){
    var files = make([]fj,0)
    var dirs = make([]fj,0)

    path := c.DefaultQuery("p", "/")
    filter := c.DefaultQuery("f","")
    sort_order := c.DefaultQuery("s","dsc")

    if validatePath(path) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, path)
        return
    }

    if filter != "" {
        re, err := regexp.Compile(filter)
        if err != nil {
            log.Printf("There is a problem with your regexp\n")
            c.String(http.StatusBadRequest, "Regexp problem")
            return
        }
        werr := fbgo.Walk(conf.BasePath + path, func(rpath string, file os.FileInfo, err error) error {
            if err != nil {
                log.Printf("prevent panic by handling failure accessing a path %q: %v\n", rpath, err)
                c.String(http.StatusBadRequest, "Search error")
                return err
            }
            if re.MatchString(file.Name()) {
//            if re.MatchString(rpath) {
                rel, _ := filepath.Rel(conf.BasePath, rpath)
                if file.IsDir() {
                    e := fj{"d", rel, file.Size()}
                    dirs = append(dirs, e)
                } else {
                    if checkFileIsImg(strings.ToLower(file.Name())) {
                        e := fj{"f", rel, file.Size()}
                        files = append(files, e)
                    }
                }

            }
            return nil
        })
        if werr != nil {
            log.Printf("error walking the path")
            c.String(http.StatusBadRequest, "Search error")
            return
        }
    } else {
        f, err := ioutil.ReadDir(conf.BasePath + path)
        if err != nil {
            c.String(http.StatusBadRequest, "Bad Request")
            return
        }
        for _, file := range f {
            if file.IsDir() {
                e := fj{"d", file.Name(), file.Size()}
                dirs = append(dirs, e)
            } else {
                if checkFileIsImg(strings.ToLower(file.Name())) {
                    e := fj{"f", file.Name(), file.Size()}
                    files = append(files, e)
                }
            }
        }
    }

    if sort_order == "dsc" {
        var tmpf = make([]fj,0)
        for i := len(files)-1; i >= 0; i-- {
            tmpf = append(tmpf,files[i])
        }
        files = tmpf
    }
    for _, v := range files {
        dirs = append(dirs, v)
    }
    c.JSON(http.StatusOK, dirs)
}

func uploadFile(c *gin.Context) {
//    c.Request.ParseForm()
//    file := c.PostForm("file")
    dstPath := c.PostForm("dst")
    if(dstPath != "/"){
        dstPath += "/"
    }

    if validatePath(dstPath) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, dstPath)
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        c.String(http.StatusBadRequest, fmt.Sprintf("Form parameter err: %s", err.Error()))
        return
    }
    filename := filepath.Base(file.Filename)
    log.Printf("Uploading %s to %s", filename, dstPath)

    if err := c.SaveUploadedFile(file, conf.BasePath + dstPath + filename); err != nil {
        c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
        return
    }
    c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully", file.Filename))
}

func apiAuth(c *gin.Context) {
            c.Request.ParseForm()
            userid := c.PostForm("user")
            password := c.PostForm("password")

            if strings.Trim(userid, " ") == "" || strings.Trim(password, " ") == "" {
                log.Println("Got empty parameters")
                c.JSON(http.StatusBadRequest, gin.H{"data": ""})
                return
            }

            user := User{}
            if err := db.Read("users", userid , &user); err != nil {
                log.Println("No such user. Error", err)
                c.JSON(http.StatusBadRequest, gin.H{"data": ""})
                return
            }
            if bcrypt.CompareHashAndPassword([]byte(user.Pass), []byte(password)) == nil && user.Name == userid {
                session := sessions.Default(c)
                session.Set("user", userid)
                err := session.Save()
                if err != nil {
                    log.Println("Failed to generate session token")
                    c.JSON(http.StatusBadRequest, gin.H{"data": ""})
                    return
                } else {
                    log.Println("Successfully authenticated user")
                    c.JSON(http.StatusOK, gin.H{"data": ""})
                    return
                }
            } else {
                    log.Println("Invalid password")
                    c.JSON(http.StatusBadRequest, gin.H{"data": ""})
                    return
            }
}
