// Package docs provides Swagger documentation for PAYMENTFC API.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "title": "PAYMENTFC API",
        "description": "Payment, Xendit invoice, webhook, and audit for Go Commerce.",
        "version": "1.0"
    },
    "host": "localhost:28083",
    "basePath": "/",
    "paths": {
        "/ping": {
            "get": {"summary": "Ping", "responses": {"200": {"description": "pong"}}}
        },
        "/health": {
            "get": {"summary": "Health", "responses": {"200": {"description": "healthy"}}}
        },
        "/v1/payment/webhook": {
            "post": {
                "summary": "Xendit webhook callback",
                "description": "Called by Xendit on payment events. Requires x-callback-token header.",
                "parameters": [
                    {"in": "header", "name": "x-callback-token", "type": "string", "required": true},
                    {"in": "body", "name": "body", "schema": {"type": "object"}}
                ],
                "responses": {"200": {"description": "processed"}, "401": {"description": "Invalid token"}}
            }
        },
        "/api/v1/payment/invoice": {
            "post": {
                "security": [{"BearerAuth": []}],
                "summary": "Create Xendit invoice",
                "parameters": [{
                    "in": "body",
                    "name": "body",
                    "schema": {
                        "type": "object",
                        "properties": {
                            "order_id": {"type": "integer"},
                            "user_id": {"type": "integer"},
                            "total_amount": {"type": "number"},
                            "payment_method": {"type": "string"},
                            "shipping_address": {"type": "string"}
                        }
                    }
                }],
                "responses": {"200": {"description": "id, invoice_url, status"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/api/v1/invoice/{order_id}/pdf": {
            "get": {
                "security": [{"BearerAuth": []}],
                "summary": "Download invoice PDF",
                "parameters": [{"in": "path", "name": "order_id", "type": "integer", "required": true}],
                "responses": {"200": {"description": "PDF file"}, "401": {"description": "Unauthorized"}}
            }
        },
        "/api/v1/failed_payments": {
            "get": {
                "security": [{"BearerAuth": []}],
                "summary": "List failed payments",
                "responses": {"200": {"description": "List of failed payments"}, "401": {"description": "Unauthorized"}}
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {"type": "apiKey", "name": "Authorization", "in": "header"}
    }
}`

func init() {
	swag.Register(swag.Name, &s{})
}

type s struct{}

func (s *s) ReadDoc() string {
	return docTemplate
}
