package service

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"log"
	"os"
	"strings"
)

func SendEmailWithSendGrid(toEmailAddress, toName, subject, plainTextContent, htmlContent string) error {
	sendgridAPIKey := os.Getenv("SENDGRID_API_KEY")
	if sendgridAPIKey == "" {
		log.Println("ADVERTENCIA: SENDGRID_API_KEY no está configurada. El correo no se enviará.")
		return fmt.Errorf("SENDGRID_API_KEY no está configurada")
	}

	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")
	if fromEmail == "" {
		log.Println("ADVERTENCIA: SENDGRID_FROM_EMAIL no está configurada. El correo no se enviará.")
		return fmt.Errorf("SENDGRID_FROM_EMAIL no está configurada")
	}

	fromName := os.Getenv("SENDGRID_FROM_NAME")
	if fromName == "" {
		fromName = "GreenPark"
	}

	from := mail.NewEmail(fromName, fromEmail)
	to := mail.NewEmail(toName, toEmailAddress)

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(sendgridAPIKey)
	response, err := client.Send(message)

	if err != nil {
		log.Printf("Error al intentar enviar correo vía SendGrid a %s: %v", toEmailAddress, err)
		return fmt.Errorf("falló el envío del correo a través de SendGrid: %w", err)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		log.Printf("Correo enviado exitosamente a %s (Asunto: %s). Estado: %d", toEmailAddress, subject, response.StatusCode)
		log.Println("Cuerpo de la respuesta de SendGrid:", response.Body)
		log.Println("Cabeceras de la respuesta de SendGrid:", response.Headers)
		return nil
	}

	log.Printf("Error al enviar correo a %s vía SendGrid. Estado: %d, Cuerpo: %s, Cabeceras: %v",
		toEmailAddress, response.StatusCode, response.Body, response.Headers)
	return fmt.Errorf("SendGrid devolvió un estado no exitoso %d: %s", response.StatusCode, response.Body)
}

func SendSMS(toNumber string, messageBody string) error {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")

	if accountSid == "" || authToken == "" || fromNumber == "" {
		log.Println("ADVERTENCIA: Las credenciales de Twilio (SID, Token o From Number) no están configuradas. El SMS no se enviará.")
		return fmt.Errorf("credenciales de Twilio no configuradas completamente")
	}

	if !strings.HasPrefix(toNumber, "+") {
		log.Printf("ADVERTENCIA: El número de destino '%s' no está en formato E.164 (debe empezar con '+'). El SMS podría fallar.", toNumber)
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   accountSid,
		Password:   authToken,
		AccountSid: accountSid,
	})

	params := &openapi.CreateMessageParams{}
	params.SetTo(toNumber)
	params.SetFrom(fromNumber)
	params.SetBody(messageBody)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("Error al enviar SMS a %s vía Twilio: %v", toNumber, err)
		return fmt.Errorf("falló el envío del SMS: %w", err)
	}

	if resp != nil && resp.Sid != nil {
		log.Printf("SMS enviado exitosamente a %s. SID del Mensaje: %s", toNumber, *resp.Sid)
	} else {
		log.Printf("SMS enviado a %s, pero no se recibió SID en la respuesta (esto es inusual si no hubo error).", toNumber)
	}

	return nil
}
