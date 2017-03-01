package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocraft/web"
	"github.com/hyperledger/fabric/core/crypto"
	"github.com/hyperledger/fabric/core/crypto/primitives"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

type AppREST struct {
}

var myLogger = logging.MustGetLogger("app")

func buildRouter() *web.Router {
	router := web.New(AppREST{})

	// Add middleware
	router.Middleware((*AppREST).SetResponseType)

	api := router.Subrouter(AppREST{}, "/api")
	api.Post("/login", (*AppREST).Login)
	api.Get("/logout", (*AppREST).Logout)
	api.Get("/islogin", (*AppREST).IsLogin)
	api.Get("/my", (*AppREST).My)

	// Add routes
	currencyRouter := api.Subrouter(AppREST{}, "/currency")
	currencyRouter.Post("/create", (*AppREST).Create)
	currencyRouter.Post("/release", (*AppREST).Release)
	currencyRouter.Post("/assign", (*AppREST).Assign)
	currencyRouter.Get("/create/check/:txid", (*AppREST).CheckCreate)
	currencyRouter.Get("/release/check/:txid", (*AppREST).CheckRelease)
	currencyRouter.Get("/assign/check/:txid", (*AppREST).CheckAssign)
	currencyRouter.Get("/:id", (*AppREST).Currency)
	currencyRouter.Get("/", (*AppREST).Currencys)

	txRouter := router.Subrouter(AppREST{}, "/tx")
	txRouter.Post("/exchange", (*AppREST).Exchange)
	txRouter.Post("/cancel", (*AppREST).Cancel)
	txRouter.Get("/exchange/check/:uuid", (*AppREST).CheckOrder)
	txRouter.Get("/cancel/check/:uuid", (*AppREST).CheckCancel)

	userRouter := router.Subrouter(AppREST{}, "/user")
	// userRouter.Post("/login", (*AppREST).Login)
	// userRouter.Get("/asset/:owner", (*AppREST).Asset)
	// userRouter.Get("/currency/:user", (*AppREST).MyCurrency)
	userRouter.Get("/tx/:user", (*AppREST).MyTxs)
	// Add not found page
	router.NotFound((*AppREST).NotFound)

	return router
}

func initConfig() {
	// Now set the configuration file
	viper.SetEnvPrefix("HYPERLEDGER")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")      // path to look for the config file in
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

func main() {
	initConfig()

	// initRedis()
	// defer client.Close()

	primitives.SetSecurityLevel("SHA3", 256)
	if err := initNVP(); err != nil {
		// myLogger.Debugf("Failed initiliazing NVP [%s]", err)
		os.Exit(-1)
	}

	crypto.Init()

	// Enable fabric 'confidentiality'
	confidentiality(false)

	// Deploy
	if err := deploy(); err != nil {
		// myLogger.Errorf("Failed deploying [%s]", err)
		os.Exit(-1)
	}

	time.Sleep(time.Second * 20)
	max := 100000

	loginChan := make(chan int, max)
	sum := 1
	err := 0
	start, end := time.Now().Unix(), time.Now().Unix()

	for i := 0; i < max; i++ {
		go func(j int) {
			// clientConn, _ := peer.NewPeerClientConnection()
			// serverClient := pb.NewPeerClient(clientConn)
			TestCurrency("t"+strconv.Itoa(j), strconv.Itoa(j), "jim", loginChan)
		}(i)
	}

	// loop1:
	for {
		select {
		case flag := <-loginChan:
			if flag == 1 {
				err++
			}
			if sum == max {
				end = time.Now().Unix()
				fmt.Println("*****************", err, end-start, float64(max)/float64(end-start))
				// break loop1
				return
			}
			sum++
		}
	}

	// 	time.Sleep(time.Second * 20)

	// 	createChan := make(chan int, max)
	// 	for i := 0; i < max; i++ {
	// 		go func(j int) {
	// 			clientConn, _ := peer.NewPeerClientConnection()
	// 			serverClient := pb.NewPeerClient(clientConn)
	// 			TestgetCurrency("t"+strconv.Itoa(j), createChan, serverClient)
	// 		}(i)
	// 	}
	// 	sum1 := 1
	// 	err1 := 0
	// 	start1, end1 := time.Now().Unix(), time.Now().Unix()

	// 	for {
	// 		select {
	// 		case flag := <-createChan:
	// 			if flag == 1 {
	// 				err1++
	// 			}
	// 			// fmt.Println(sum1)
	// 			if sum1 == max {
	// 				end1 = time.Now().Unix()
	// 				fmt.Println("*****************", err, end-start, float64(max)/float64(end-start))
	// 				fmt.Println("*****************", err1, end1-start1, float64(max)/float64(end1-start1))
	// 				return
	// 			}
	// 			sum1++
	// 		}
	// 	}

	// go eventListener(chaincodeName)

	// go lockBalance()

	// go matchTx()

	// go execTx()

	// go findExpired()

	// go execExpired()

	// go execCancel()

	restAddress := viper.GetString("app.rest.address")
	tlsEnable := viper.GetBool("app.tls.enabled")

	// Initialize the REST service object
	// myLogger.Infof("Initializing the REST service on %s, TLS is %s.", restAddress, (map[bool]string{true: "enabled", false: "disabled"})[tlsEnable])

	router := buildRouter()

	// Start server
	if tlsEnable {
		err := http.ListenAndServeTLS(restAddress, viper.GetString("app.tls.cert.file"), viper.GetString("app.tls.key.file"), router)
		if err != nil {
			// myLogger.Errorf("ListenAndServeTLS: %s", err)
		}
	} else {
		err := http.ListenAndServe(restAddress, router)
		if err != nil {
			// myLogger.Errorf("ListenAndServe: %s", err)
		}
	}
}
