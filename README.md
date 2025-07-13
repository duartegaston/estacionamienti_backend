# Parking System API

This project provides a backend API for managing parking reservations, pricing, availability, and vehicle configurations for a parking facility. It is designed to be used by both administrators and end users (customers) via a web or mobile frontend.

## Purpose
- Enable users to check availability, view prices, and make/cancel reservations for parking spots.
- Allow administrators to manage reservations, configure vehicle spaces, and oversee parking operations.
- Integrate with Stripe for secure online payments.
- Enforce business rules such as cancellation windows and reservation validation.

## Key Features
- User and admin authentication (JWT-based).
- Reservation management: create, view, cancel, and list reservations.
- Real-time availability and pricing queries.
- Vehicle type and parking space configuration.
- Stripe payment integration for secure transactions.
- Custom error handling with specific HTTP status codes for business rule enforcement.

## Technologies Used
- Go (Golang) for backend API
- Gorilla Mux for HTTP routing
- Stripe Go SDK for payment processing
- PostgreSQL (assumed) for data storage