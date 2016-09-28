/*
* @Author: Jim Weber
* @Date:   2016-09-28 12:40:15
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-09-28 14:52:53
 */

package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	vaultHost := flag.String("vault", "", "Hostname of Vault Server")
	creds := flag.String("creds", "", "Vault Path to Credentials")
	mqHost := flag.String("db", "", "Hostname of MQ Server")
	clientToken := flag.String("token", "", "Your Client Token")
	flag.Parse()

	// init vault client config
	httpClient := &http.Client{}
	clientConfig := vaultapi.Config{
		Address:    "https://" + *vaultHost + ":8200",
		HttpClient: httpClient,
		MaxRetries: 3,
	}

	// intialize vault client
	client, err := vaultapi.NewClient(&clientConfig)
	if err != nil {
		log.Println(err)
	}

	log.Println("Reading MQ creds from vault")
	// don't forget by default the vault token to auth with is
	// read from your env vars. It looks for VAULT_TOKEN
	client.SetToken(*clientToken)
	secret, err := client.Logical().Read(*creds)
	if err != nil {
		log.Println(err)
	}

	username := secret.Data["username"].(string)
	password := secret.Data["password"].(string)
	log.Println("Username:", username)
	log.Println("Password:", password)

	conn, err := amqp.Dial("amqp://" + username + ":" + password + "@" + *mqHost + ":5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	for {
		body := "hello"
		err = ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
		log.Printf(" [x] Sent %s", body)
		failOnError(err, "Failed to publish a message")
		time.Sleep(time.Second * 5)
	}
}
