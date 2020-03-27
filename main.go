package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/gocolly/colly"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

type vsebina struct {
	predmet  string
	profesor string
}
type izbranRazred struct {
	izbranrazred string
}

func main() {

	//hashmap with string index and string value
	razredi := make(map[string]string)
	c := colly.NewCollector()
	c.OnHTML("#id_parameter", func(e *colly.HTMLElement) {
		e.ForEach("option", func(opt int, option *colly.HTMLElement) {
			razredi[option.Text] = option.Attr("value")
		})
	})
	//get class i want from app
	http.HandleFunc("setClass", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var razred izbranRazred
		err := decoder.Decode(&razred)
		if err != nil {
			log.Fatalln(err)
		}
		getschedule(razredi[razred.izbranrazred])
	})

	c.Visit("https://www.easistent.com/urniki/5738623c4f3588f82583378c44ceb026102d6bae/razredi/242982")
	http.ListenAndServe(":80", nil)
}

func getschedule(razred string) {
	opt := option.WithCredentialsFile("path/to/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}
	ctx := context.Background()
	client, err := app.Messaging(ctx)
	urnik := []vsebina{}
	dnevi := [9][6]vsebina{}
	c := colly.NewCollector()
	c.OnHTML("table.ednevnik-seznam_ur_teden", func(e *colly.HTMLElement) {
		e.ForEach("table.ednevnik-seznam_ur_teden > tbody > tr", func(indextr int, tr *colly.HTMLElement) {
			tr.ForEach("table.ednevnik-seznam_ur_teden > tbody > tr > td", func(indextd int, td *colly.HTMLElement) {
				predmet := td.DOM.Find(".text14").Text()
				profesor := td.DOM.Find(".text11").Text()
				neki := vsebina{strings.TrimSpace(predmet), strings.TrimSpace(profesor)}
				urnik = append(urnik, neki)
				//fmt.Println(strings.TrimSpace(neki.predmet))
				//fmt.Println(strings.TrimSpace(neki.profesor))
				//fmt.Println(indextd)
				dnevi[indextr][indextd] = neki
			})
		})
		//fmt.Println(dnevi)

		for ura := 0; ura < 11; ura++ {
			for dan := 0; dan < 6; dan++ {
				fmt.Print(dnevi[ura][dan])
			}
			fmt.Println()
		}

		ure := []string{"7.05", "7.50", "8.40", "9.30", "10.20", "11.10", "12.00", "12.50", "13.40"}
		danasnjiDan := int(time.Now().Weekday()) + 1
		for i := 1; i < 11; i++ {
			stringToTime, _ := time.Parse("15.04", ure[i])
			timeDIFF := stringToTime.Sub(time.Now())
			time.AfterFunc(timeDIFF, func() {
				topic := "notification"
				message := &messaging.Message{
					Data: map[string]string{
						"imePredmeta": dnevi[i][danasnjiDan].predmet,
						"profesor":    dnevi[i][danasnjiDan].profesor,
					},
					Topic: topic,
				}
				// Send a message to the devices subscribed to the provided topic.
				response, err := client.Send(ctx, message)
				if err != nil {
					log.Fatalln(err)
				}
				// Response is a message ID string.
				fmt.Println("Successfully sent message:", response)

			})
		}
	})
	//set class i want to get schedule from
	c.Visit("https://www.easistent.com/urniki/5738623c4f3588f82583378c44ceb026102d6bae/razredi/" + razred)

}
