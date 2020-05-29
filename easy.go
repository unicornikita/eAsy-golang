package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/gocolly/colly"
	"google.golang.org/api/option"
)

type vsebina struct {
	Predmet  string
	Profesor string
	StPredmetov int
}

var dnevi [9][6]vsebina = [9][6]vsebina{}

var client *messaging.Client
var ctx context.Context

func main() {
	ctx = context.Background()
	opt := option.WithCredentialsFile("/home/pi/Documents/easy-matura-config.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}
	if app == nil {
		fmt.Println("app is nil")
	}

	client, err = app.Messaging(ctx)

	//hashmap with string index and string value
	razredi := make(map[string]string)
	c := colly.NewCollector()
	c.OnHTML("#id_parameter", func(e *colly.HTMLElement) {
		e.ForEach("option", func(opt int, option *colly.HTMLElement) {
			razredi[option.Text] = option.Attr("value")
		})

	})
	//get class i want from app
	//schedule for current day
	http.HandleFunc("/danes/", func(w http.ResponseWriter, r *http.Request) {

		razred := strings.TrimPrefix(r.URL.Path, "/danes/")
		getschedule(razredi[razred])

		indexDneva := int(time.Now().Weekday())
		if int(time.Now().Weekday()) == 0 || int(time.Now().Weekday()) == 6 {
			indexDneva = 1
		}

		var urnikDanes [9]vsebina = [9]vsebina{}
		for i := 0; i < 9; i++ {
			urnikDanes[i] = dnevi[i][indexDneva]
		}
		sendData(w, r, urnikDanes)
	})
	//schedule for selected other day
	http.HandleFunc("/izbranDan/", func(w http.ResponseWriter, r *http.Request) {
		podatki := strings.TrimPrefix(r.URL.Path, "/izbranDan/")
		izbranRazred := strings.Split(podatki, "/")[0]
		izbranDan, err := strconv.Atoi(strings.Split(podatki, "/")[1])
		if err != nil {
			log.Fatal(err)
		}
		getschedule(razredi[izbranRazred])

		var izbranUrnik [9]vsebina = [9]vsebina{}
		for i := 0; i < 9; i++ {
			izbranUrnik[i] = dnevi[i][izbranDan]
		}
		sendData(w, r, izbranUrnik)
	})
	//send all classes to app
	http.HandleFunc("/allClasses", func(w http.ResponseWriter, r *http.Request) {
		imenarazredov := []string{}
		for k := range razredi {
			imenarazredov = append(imenarazredov, k)
		}
		b, err := json.Marshal(imenarazredov)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprint(w, string(b))
	})

	c.Visit("https://www.easistent.com/urniki/5738623c4f3588f82583378c44ceb026102d6bae/razredi/242982")
	fmt.Println("listening on port 443")
	log.Fatal(http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/easy-matura.ddns.net/fullchain.pem", "/etc/letsencrypt/live/easy-matura.ddns.net/privkey.pem", nil))
}

func getschedule(razred string) {
	urnik := []vsebina{}
	c := colly.NewCollector()
	c.OnHTML("table.ednevnik-seznam_ur_teden", func(e *colly.HTMLElement) {
		e.ForEach("table.ednevnik-seznam_ur_teden > tbody > tr", func(indextr int, tr *colly.HTMLElement) {
			tr.ForEach("table.ednevnik-seznam_ur_teden > tbody > tr > td", func(indextd int, td *colly.HTMLElement) {
				predmet := td.DOM.Find(".text14").Text()
				tr.DOM.Children
				if numPredmetov > 1 {
					fmt.Println("Predmetov je")
					fmt.Println(numPredmetov)
				}

				numPredmetov = 0
				profesor := td.DOM.Find(".text11").Text()
				prebraniPodatki := vsebina{strings.TrimSpace(predmet), strings.TrimSpace(profesor)}
				urnik = append(urnik, prebraniPodatki)
				//fmt.Println(predmet)
				//fmt.Println(profesor)
				//fmt.Println(indextd)
				dnevi[indextr-1][indextd] = prebraniPodatki
			})
		})
		//fmt.Println(dnevi)

		/*for ura := 0; ura < 9; ura++ {
			for dan := 0; dan < 6; dan++ {
				fmt.Print(dnevi[ura][dan])
			}
			fmt.Println()
		}*/
	})

	go sendToFirebase()
	//set class i want to get schedule from
	c.Visit("https://www.easistent.com/urniki/5738623c4f3588f82583378c44ceb026102d6bae/razredi/" + razred)

}

func sendData(w http.ResponseWriter, r *http.Request, data [9]vsebina) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, string(b))
	fmt.Print(string(b))
}

func sendToFirebase() {
	ure := []string{"7.00", "7.50", "8.40", "9.30", "10.20", "11.10", "12.00", "12.50", "13.40"}
	danasnjiDan := int(time.Now().Weekday())
	if int(time.Now().Weekday()) == 0 || int(time.Now().Weekday()) == 6 {
		danasnjiDan = 1
	}
	fmt.Println(danasnjiDan)
	for i := 1; i < 9; i++ {
		fmt.Println(i)
		stringToTime, _ := time.Parse("15.04", ure[i])
		timeDIFF := stringToTime.Sub(time.Now())
		go func(j int) {
			time.Sleep(timeDIFF)
			imePredmeta := dnevi[j][danasnjiDan].Predmet
			profesor := dnevi[j][danasnjiDan].Profesor
			message := &messaging.Message{
				Notification: &messaging.Notification{
					Title: imePredmeta,
					Body:  profesor,
				},
				Topic: "notification",
			}
			// Send a message to the devices subscribed to the provided topic.
			response, err := client.Send(ctx, message)
			if err != nil {
				log.Fatalln(err)
			}

			// Response is a message ID string.
			fmt.Println("Successfully sent message:", response)
		}(i)

	}
}
