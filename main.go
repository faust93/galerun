package main

import (
    "log"
    "errors"
    "regexp"
    "fmt"
    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/cookie"
    "github.com/gin-gonic/gin"
    "gopkg.in/h2non/bimg.v1"
    "net/http"
    "os"
    "strings"
    "path/filepath"
    "io"
    "io/ioutil"
    "flag"
    "strconv"
    "encoding/json"
    "golang.org/x/crypto/bcrypt"
    "github.com/tucnak/store"
    "github.com/rwcarlsen/goexif/exif"
    scribble "github.com/nanobox-io/golang-scribble"
    fbgo "github.com/facebookgo/symwalk"
)

var conf Config
var db *scribble.Driver

type Config struct {
    BasePath string `toml:"imagesRoot"`
    AppPath string `toml:"appRoot"`
    ListenTo string `toml:"serverAddr"`
    ListenPort int `toml:"serverPort"`
    ThumbSize []int `toml:"thumbSizes"`
    ScaleFactor []float64 `toml:"scaleFactors"`
    UseSSL bool `toml:"useSSL"`
    SSLCert string `toml:"SSLcert"`
    SSLKey string `toml:"SSLkey"`
}

type Pref struct {
    ThumbS int
    ScaleF float64
}

type User struct {
    Name string
    Pass string
    Role int
    Prefs Pref
}

type fj struct {
    Type string `json:"type"`
    Name string `json:"name"`
    Size int64 `json:"size"`
}

func validatePath(path string) bool {
    match, _ := regexp.MatchString(`\.\.`, path)
    return match
}

func isFileExist(file string)(bool,bool) {
    fi, err := os.Stat(file)
    if os.IsNotExist(err) {
        log.Printf("File does not exists: %s", file)
        return false,false
    }
    if fi.Mode().IsDir() {
        return false,true
    }
    return true,true
}

// Get thumbnail of specific image
// In case of folder - first image within folder
func getThumbnail(c *gin.Context) {
    img := c.Query("f")
    w := c.DefaultQuery("w", "128")
    h := c.DefaultQuery("h", "128")

    width, err := strconv.Atoi(w)
    if err != nil {
        log.Printf("Error: %s", err)
        width = 200
    }
    height, err := strconv.Atoi(h)
    if err != nil {
        log.Printf("Error: %s", err)
        height = 200
    }

    if validatePath(img) {
        log.Println("%s Path traversal attempt: %s", c.Request.RemoteAddr, img)
        return
    }
    var options bimg.Options
    var img_path string
    isF, exist := isFileExist(conf.BasePath + img)
    if exist {
        if isF {
            options = bimg.Options{
                Width: width,
                Height: height,
                Crop: true,
                Quality: 65,
                Type: bimg.WEBP,
                Interpolator: bimg.Nearest,
                StripMetadata: true,
                Interlace: true,
            }
            img_path = conf.BasePath + img
        } else {
            f := getFirstImg(conf.BasePath + img)
            if f == "" {
                img_path = "assets/folder.png"
            } else {
                img = img + f
                img_path = conf.BasePath + img
            }
            options = bimg.Options{
                Width: width,
                Height: height,
                Crop: true,
                Quality: 65,
                Type: bimg.WEBP,
                Interpolator: bimg.Nearest,
                Interpretation: bimg.InterpretationGREY16,
                StripMetadata: true,
                Interlace: true,
                }
        }
        pix, _ := bimg.Read(img_path)
        newPix, err := bimg.Resize(pix,options)
        if err != nil {
         log.Printf("Error: %s", err)
        }
        c.Writer.Write(newPix)
    } else {
        c.String(http.StatusBadRequest, "No such file")
    }
}

func getVideo(c *gin.Context) {
    log.Println(c.Request.Header)
    vid := c.Query("f")
    if validatePath(vid) {
        log.Printf("%s Path traversal attempt: %s", c.Request.RemoteAddr, vid)
        return
    }
    if isf,_ := isFileExist(conf.BasePath + vid); isf {
          c.File(conf.BasePath + vid)
    }
}

func getImage(c *gin.Context) {
    img := c.Query("f")

    if validatePath(img) {
        log.Println("%s Path traversal attempt: %s", c.Request.RemoteAddr, img)
        return
    }

    //scalefactor
    s := c.DefaultQuery("s", "0")
    scale, err := strconv.ParseFloat(s,64)
    if err != nil {
        log.Printf("Error: %s", err)
        scale = 0
    }

    isf, exist := isFileExist(conf.BasePath + img)
    if exist && isf {
        pix, _ := bimg.Read(conf.BasePath + img)
        size, err := bimg.NewImage(pix).Size()
        if err != nil {
            log.Printf("Error: %s", err)
        }
        if scale > 0 && (size.Height > 2048 || size.Width > 2048) {
            w, _ := strconv.Atoi(fmt.Sprintf("%.0f", float64(size.Width)/scale))
            h, _ := strconv.Atoi(fmt.Sprintf("%.0f", float64(size.Height)/scale))
            newPix, err := bimg.NewImage(pix).Resize(w, h)
            if err != nil {
                log.Printf("Error: %s", err)
            }
            c.Writer.Write(newPix)
        } else {
            c.Writer.Write(pix)
        }
    } else {
        c.String(http.StatusBadRequest, "No such file")
    }
}

// http
func getVideoList(fpath string) []string {
    var s []string
    err := fbgo.Walk(fpath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            log.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
            return err
        }
        if !info.IsDir() {
            if checkFileIsVid(info.Name()) {
            rel, _ := filepath.Rel(fpath,path)
            s = append(s, "/"+rel)
            }
        }
        return nil
      })
      if err != nil {
        log.Printf("error walking the path")
        return nil
        }
     return s
}

func getFirstImg(fpath string) string {
    var s string
    err := fbgo.Walk(fpath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            log.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
            return err
        }
        if !info.IsDir() {
            if checkFileIsImg(info.Name()) {
            rel, _ := filepath.Rel(fpath,path)
            s = "/" + rel
            return io.EOF
            }
        }
        return nil
      })
      if err != io.EOF {
        log.Printf("error walking the path")
        return ""
        }
     return s
}

func getDirList(fpath string) []fj {
    var files = make([]fj,0)
    var t string
    var size int64

    f, err := ioutil.ReadDir(conf.BasePath + fpath)
    if err != nil {
        log.Printf("Wrong path " + fpath)
        return nil
    }

    for _, file := range f {
        if file.IsDir() {
            t = "d"
            n, _ := ioutil.ReadDir(conf.BasePath + fpath + "/" + file.Name())
            size = int64(len(n))
        } else {
            t = "f"
            size = file.Size()
        }

        if (t == "f" && checkFileIsImg(strings.ToLower(file.Name()))) || t == "d" {
            e := fj{t, file.Name(), size}
            files = append(files, e)
        }
    }
    return files
}

func fetchExif(filename string) (string, error) {
    var out strings.Builder

    f, err := os.Open(conf.BasePath + filename)
    if err != nil {
        log.Println("EXIF",err)
        return "", errors.New("Unable to open " + filename)
    }
    defer f.Close()

    x, err := exif.Decode(f)
    if err != nil {
        log.Println("EXIF", err)
        return "", errors.New("Unable to decode EXIF")
    }

    lat, long, _ := x.LatLong()
    iso, _ := x.Get(exif.ISOSpeedRatings)
    w, _ := x.Get(exif.PixelXDimension)
    h, _ := x.Get(exif.PixelYDimension)
    exp, _ := x.Get(exif.ExposureTime)
    mod, _ := x.Get(exif.Model)
    dt, _ := x.Get(exif.DateTimeOriginal)

    if dt != nil {
        s := fmt.Sprintf("Date: %s<br>", dt);
        out.WriteString(s)
    }
    if w != nil && h != nil {
        s := fmt.Sprintf("Image size: %sx%s<br>", w,h);
        out.WriteString(s)
    }
    if mod != nil {
        s := fmt.Sprintf("Model: %s<br>", mod);
        out.WriteString(s)
    }
    if iso != nil {
        s := fmt.Sprintf("ISO: %s<br>", iso);
        out.WriteString(s)
    }
    if exp != nil {
        s := fmt.Sprintf("Exposure: %s<br>", exp);
        out.WriteString(s)
    }
    if lat != 0 && long != 0 {
        s := fmt.Sprintf("Lat, Long: %.2f %.2f<br>", lat,long);
        out.WriteString(s)
    }

    return out.String(), nil
}

func checkFileIsVid(fn string) bool {
    fileTypes := []string{".mp4", ".webm", ".ogv", ".ogg"}
    result := false
    for _, f := range fileTypes {
        if strings.HasSuffix(fn, f) {
            result = true
            break
        }
    }
    return result
}

func checkFileIsImg(fn string) bool {
    fileTypes := []string{".jpeg", ".jpg", ".gif", ".png", ".webp"}
    result := false
    for _, f := range fileTypes {
        if strings.HasSuffix(fn, f) {
            result = true
            break
        }
    }
    return result
}

func ByteConvert(b int64) string {
    const unit = 1000
    if b < unit {
        return fmt.Sprintf("%d B", b)
    }
    div, exp := int64(unit), 0
    for n := b / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}


var r *gin.Engine

func main() {
    var err error
    db, err = scribble.New("./conf", nil)
    if err != nil {
        log.Println("Error", err)
        return
    }

// args parser
    if len(os.Args) > 1 {
        if os.Args[1] == "user" && len(os.Args) > 2 {
            switch os.Args[2] {
                case "create":
                createCmd := flag.NewFlagSet("create", flag.ExitOnError)
                userNamePtr := createCmd.String("username", "", "User name to create")
                userPassPtr := createCmd.String("password", "", "User password")
                userRolePtr := createCmd.Int("role", 1, "User role: 0 - rw, 1 - readonly")

                createCmd.Parse(os.Args[3:])
                if createCmd.Parsed() && *userNamePtr != "" && *userPassPtr != "" {
                    hash, err := bcrypt.GenerateFromPassword([]byte(*userPassPtr), bcrypt.DefaultCost)
                    if err != nil {
                        log.Fatal(err)
                    }
                    if err := db.Write("users", *userNamePtr, User{Name: *userNamePtr, Role: *userRolePtr, Pass: string(hash), Prefs: Pref{ThumbS: 128, ScaleF: 128}}); err != nil {
                        log.Fatal(err)
                    }
                    fmt.Println(*userNamePtr + " has been created")
                } else {
                    createCmd.PrintDefaults()
                }
                return
                case "list":
                    records, err := db.ReadAll("users")
                    if err != nil {
                        log.Fatal(err)
                    }
                    fmt.Println("User\tRole")
                    for _, f := range records {
                         user := User{}
                         if err := json.Unmarshal([]byte(f), &user); err != nil {
                            log.Fatal(err)
                         }
                        fmt.Printf("%s\t%d\n", user.Name, user.Role )
                    }
                    return
                case "delete":
                    delCmd := flag.NewFlagSet("delete", flag.ExitOnError)
                    userNamePtr := delCmd.String("username", "", "User name to remove")
                    delCmd.Parse(os.Args[3:])
                    if delCmd.Parsed() && *userNamePtr != "" {
                        if err := db.Delete("users", *userNamePtr); err != nil {
                            log.Fatal(err)
                        } else {
                          fmt.Println(*userNamePtr + " has been removed")
                        }
                    } else {
                        delCmd.PrintDefaults()
                    }
                    return
                default:
                    fmt.Println("Unknown argument")
                    return
            }
        } else {
            fmt.Println("Accepted arguments are: user")
            fmt.Println("'user' requires an additional parameter:\n 'create' - create an user:\n\t-username - user name\n\t-password - user password\n\t-role - user role (0 rw, 1 readonly)")
            fmt.Println("'delete' - remove an user:\n\t-username - user name")
            fmt.Println("'list' - show existing users")
            return
        }
    }

    store.Init("conf")
    if err := store.Load("server.toml", &conf); err != nil {
        log.Println("failed to load server config:", err)
        return
    }

//    gin.SetMode(gin.ReleaseMode)
    gin.ForceConsoleColor()

    r = gin.New()
    store := cookie.NewStore([]byte("secret"))
    store.Options(sessions.Options{
        MaxAge: 1800,
    })
    r.Use(sessions.Sessions("mysession", store))

    r.MaxMultipartMemory = 8 << 20 // 8 MiB
    r.Use(gin.Recovery())
    r.Use(gin.Logger())

    r.Static("/assets", "./assets")
    r.LoadHTMLGlob("templates/*")

    initRoutes()

    if conf.UseSSL == true {
        r.RunTLS(conf.ListenTo + ":" + strconv.Itoa(conf.ListenPort), conf.SSLCert, conf.SSLKey)
    } else {
        r.Run(conf.ListenTo + ":" + strconv.Itoa(conf.ListenPort))
    }
}