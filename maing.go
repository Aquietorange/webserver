package main

import (
	"context"
	"fmt"
	fileserve "htmlserve/FileServe"
	conf "htmlserve/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var config conf.Config
var servers sync.Map

func myHandler(res http.ResponseWriter, req *http.Request) {
	//如需跨域 需添加以下 head 字段
	res.Header().Add("access-control-allow-credentials", "true")
	res.Header().Add("access-control-allow-origin", "*")
	res.Header().Add("access-control-allow-methods", "POST, GET, OPTIONS")

	//如请求中有其它自定义Heade字段，需在下面添加
	res.Header().Add("Access-Control-Allow-Headers", "Authorization,x-requested-with,content-type")
	res.Header().Add("access-control-max-age", "86400")

	fmt.Println("url:", req.URL)
	fmt.Println("header:", req.Header)
	http.SetCookie(res, &http.Cookie{
		Name:  "iid",
		Value: "88999",
	})
	res.Write([]byte("ok"))
}

func main() {
	if len(config.Serves) > 0 {

		for _, s := range config.Serves {

			go startserve(s)

		}
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
}

func startserve(s conf.Serve) {

	srv := &http.Server{Addr: ":" + strconv.Itoa(s.Port), Handler: fileserve.MyFileServer(http.Dir(s.Path), s)}

	//mux := http.NewServeMux()

	//	mux.HandleFunc("/testh", myHandler)
	//mux.Handle("/", fileserve.MyFileServer(http.Dir(s.Path), s))

	fmt.Println("正在启动web服务器，端口:" + strconv.Itoa(s.Port) + ";path:" + s.Path)
	//err := http.ListenAndServe(":"+strconv.Itoa(s.Port), mux)
	servers.Store(s.Path+strconv.Itoa(s.Port), srv)
	err := srv.ListenAndServe()

	if err != nil {
		servers.Delete(s.Path + strconv.Itoa(s.Port))
		log.Println("启动失败或重启停止端口" + strconv.Itoa(s.Port))
		return
	}

}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panicln(err)
		log.Fatal("read config failed")
	}
	//ss := viper.Get("serve")
	viper.Unmarshal(&config)
	fmt.Println(config.Serves)

	viper.WatchConfig() //监听配置变化
	ctx := context.Background()

	viper.OnConfigChange(func(e fsnotify.Event) { //配置变化回调
		fmt.Printf("Config file:%s Op:%s\n", e.Name, e.Op)
		viper.Unmarshal(&config) //需重新绑定 配置
		fmt.Println(config)

		var hamp = map[string]bool{}
		for _, s := range config.Serves {
			hamp[s.Path+strconv.Itoa(s.Port)] = true
		}

		servers.Range(func(key, value interface{}) bool { //停止已修改的serve
			v, ok := hamp[key.(string)]
			if !ok || !v { //
				var s = value.(*http.Server)
				timeoutCtx, _ := context.WithTimeout(ctx, 2*time.Second)
				log.Println("停止端口:" + key.(string))
				s.Shutdown(timeoutCtx)
				servers.Delete(key)
			}
			return true
		})

		for _, s := range config.Serves { ////启动新serve

			v, ok := servers.Load(s.Path + strconv.Itoa(s.Port))

			if v == nil || !ok {
				go startserve(s)
			}

		}

	})

}
