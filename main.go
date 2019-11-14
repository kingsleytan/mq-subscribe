package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v3"
	"github.com/streadway/amqp"
)

type input struct {
	ID           string `json:"id" validate:"required"`
	To           string `json:"to" validate:"required"`
	From         string `json:"from"`
	Domain       string `json:"domain"`
	Subject      string `json:"subject"`
	TemplateData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"templateData" validate:"required"`
	Template    string `json:"template" validate:"required"`
	ReferenceID string `json:"referenceID"`
	Status      string `json:"status" validate:"required"`
	Events      string `json:"events"`
}

type output struct {
	To           string `json:"to"`
	From         string `json:"from"`
	Domain       string `json:"domain"`
	TemplateData struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"templateData" validate:"required"`
	Template    string `json:"template"`
	ReferenceID string `json:"referenceID"`
}

func main() {
	var i *input
	conn, err := amqp.Dial("amqp://setucdwc:8HPqKaOisQhptp7HARM0S1rUaQeAw2LU@cougar.rmq.cloudamqp.com/setucdwc")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"forgot-password-email", // name
		true,                    // durable
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			fmt.Println("string:", string(d.Body))
			err := json.Unmarshal(d.Body, &i)
			if err != nil {
				return
			}
			sendEmail(i)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	return
}

// func init(){
// 	fmt.Println("os.GetEnv", os.Getenv("DOMAIN"))
// }

func sendEmail(m *input) {
	templateName := m.Template
	mg := mailgun.NewMailgun(os.Getenv("DOMAIN"), os.Getenv("MAILGUN_KEY"))

	var err error

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Create a new message with template
	msg := mg.NewMessage(m.From, m.Subject, "")
	msg.SetTemplate(templateName)

	// Add recipients
	msg.AddRecipient(m.To)

	// Add the variables to be used by the template
	msg.AddVariable("title", m.TemplateData.Title)
	msg.AddVariable("body", m.TemplateData.Body)

	_, id, err := mg.Send(ctx, msg)
	result := fmt.Sprintf("Queued: %s", id)
	fmt.Println("Result:", result)
	if err != nil {
		return
	}

	// var r *output
	// r.To = m.To
	// r.From = m.From
	// r.Domain = m.Domain
	// r.TemplateData.Title = m.TemplateData.Title
	// r.TemplateData.Body = m.TemplateData.Body
	// r.Template = m.Template
	// r.ReferenceID = m.ReferenceID

	// return map[string]interface{}{
	// 	"item":   r,
	// 	"result": fmt.Sprintf("result: %s", result),
	// }
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
