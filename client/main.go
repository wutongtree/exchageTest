package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	logging "github.com/op/go-logging"

	"time"

	_ "github.com/wutongtree/exchange/client/bootstrap"
	"github.com/wutongtree/exchange/client/models"
	_ "github.com/wutongtree/exchange/client/routers"
)

// var config
var (
	logger = logging.MustGetLogger("exchange.client")
)

func writeHyperledgerExplorer() {
	hyperledger_explorer := beego.AppConfig.String("hyperledger_explorer")
	filename := "static/explorer/hyperledger.js"
	fout, err := os.Create(filename)
	defer fout.Close()

	if err != nil {
		fmt.Printf("Write hyperledger exploer error: %v\n", err)
	} else {
		content := fmt.Sprintf("const REST_ENDPOINT = \"%v\";", hyperledger_explorer)
		fout.WriteString(content)
		fmt.Printf("Write hyperledger explorer with: %v\n", hyperledger_explorer)
	}
}

func main() {
	max := 5000
	// 	loginChan := make(chan int)

	// 	for i := 0; i < max; i++ {
	// 		go models.Login("jim", "6avZQLwcUe9b", loginChan)
	// 	}
	// 	sum := 1
	// 	err := 0
	// 	start, end := time.Now().Unix(), time.Now().Unix()

	// loop1:
	// 	for {
	// 		select {
	// 		case flag := <-loginChan:
	// 			if flag == 1 {
	// 				err++
	// 			}
	// 			fmt.Println(sum)
	// 			if sum == max {
	// 				end = time.Now().Unix()
	// 				fmt.Println("*****************", err, end-start, float64(max)/float64(end-start))
	// 				break loop1
	// 			}
	// 			sum++
	// 		}
	// 	}

	// createChan := make(chan int)
	// for i := 0; i < max; i++ {
	// 	go func(j int) {
	// 		currency := &models.Currency{
	// 			ID:    "testt" + strconv.Itoa(j),
	// 			Count: 100,
	// 			User:  "jim",
	// 		}
	// 		models.CreateCurrency(currency, createChan)
	// 	}(i)
	// }
	// sum1 := 1
	// err1 := 0
	// start1, end1 := time.Now().Unix(), time.Now().Unix()

	// for {
	// 	select {
	// 	case flag := <-createChan:
	// 		if flag == 1 {
	// 			err1++
	// 		}
	// 		fmt.Println(sum1)
	// 		if sum1 == max {
	// 			end1 = time.Now().Unix()
	// 			// fmt.Println("*****************", err, end-start, float64(max)/float64(end-start))
	// 			fmt.Println("*****************", err1, end1-start1, float64(max)/float64(end1-start1))
	// 			return
	// 		}
	// 		sum1++
	// 	}
	// }

	createChan := make(chan int)
	for i := 0; i < max; i++ {
		go func() {
			models.CurrencyId("testt1", createChan)
		}()
	}
	sum1 := 1
	err1 := 0
	start1, end1 := time.Now().Unix(), time.Now().Unix()

	for {
		select {
		case flag := <-createChan:
			if flag == 1 {
				err1++
			}
			fmt.Println(sum1)
			if sum1 == max {
				end1 = time.Now().Unix()
				// fmt.Println("*****************", err, end-start, float64(max)/float64(end-start))
				fmt.Println("*****************", err1, end1-start1, float64(max)/float64(end1-start1))
				return
			}
			sum1++
		}
	}

	// beego.SetStaticPath("/static", "static")
	// beego.BConfig.WebConfig.DirectoryIndex = true
	// // Write hyperledger explorer config
	// writeHyperledgerExplorer()

	// beego.InsertFilter("/*", beego.BeforeRouter, filterUser)
	// beego.ErrorHandler("404", pageNotFound)
	// beego.ErrorHandler("401", pageNoPermission)
	// beego.Run()
}

var filterUser = func(ctx *context.Context) {
	_, ok := ctx.Input.Session("userLogin_exchange").(string)
	if !ok && ctx.Request.RequestURI != "/login" {
		ctx.Redirect(302, "/login")
	}
}

func pageNotFound(rw http.ResponseWriter, r *http.Request) {
	t, _ := template.New("404.tpl").ParseFiles("views/404.tpl")
	data := make(map[string]interface{})
	t.Execute(rw, data)
}

func pageNoPermission(rw http.ResponseWriter, r *http.Request) {
	t, _ := template.New("401.tpl").ParseFiles("views/401.tpl")
	data := make(map[string]interface{})
	t.Execute(rw, data)
}
