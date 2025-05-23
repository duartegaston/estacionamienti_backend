# Parking System API Documentation

## Authentication Endpoints

### Admin Authentication
- **POST /admin/login**
  - Authenticates an admin user
  - Request Body:
    ```json
    {
      "user": "string",
      "password": "string"
    }
    ```
  - Response: JWT token for authentication

## Admin Endpoints
All admin endpoints require JWT authentication.

### Reservations Management
- **GET /admin/reservations**
  - List all reservations
  - Query Parameters:
    - `date`: Filter by date
    - `vehicle_type`: Filter by vehicle type
    - `status`: Filter by reservation status
  - Response: Array of reservations

- **POST /admin/reservations**
  - Create a new reservation (admin)
  - Request Body: Reservation details

- **DELETE /admin/reservations/{code}**
  - Delete/Cancel a reservation
  - Path Parameters:
    - `code`: Reservation code
  - Response: Success message

### Vehicle Configuration
- **GET /admin/vehicle-config**
  - List vehicle spaces configuration
  - Response: Array of vehicle spaces

- **PUT /admin/vehicle-config/{vehicle_type}**
  - Update vehicle spaces for a specific type
  - Path Parameters:
    - `vehicle_type`: Type of vehicle
  - Request Body:
    ```json
    {
      "total_spaces": "integer",
      "available_spaces": "integer"
    }
    ```

## User Endpoints

### Reservations
- **POST /reservations**
  - Create a new reservation
  - Request Body:
    ```json
    {
      "user_name": "string",
      "user_email": "string",
      "user_phone": "string",
      "vehicle_type": "string",
      "vehicle_plate": "string",
      "vehicle_model": "string",
      "payment_method": "string",
      "start_time": "datetime",
      "end_time": "datetime"
    }
    ```
  - Response:
    ```json
    {
      "reservation_code": "string",
      "message": "Reservation confirmed."
    }
    ```

- **GET /reservations/{code}**
  - Get reservation details
  - Path Parameters:
    - `code`: Reservation code
  - Request Body:
    ```json
    {
      "email": "string"
    }
    ```
  - Response: Reservation details

### Pricing and Availability
- **GET /prices**
  - Get pricing for all vehicle types and durations
  - Response: Array of prices by vehicle type and duration

- **GET /availability**
  - Check space availability
  - Query Parameters:
    - Vehicle type
    - Start time
    - End time
  - Response: Available spaces

## Common Response Formats

### Reservation Object
```json
{
  "code": "string",
  "user_name": "string",
  "user_email": "string",
  "user_phone": "string",
  "vehicle_type": "string",
  "vehicle_plate": "string",
  "vehicle_model": "string",
  "payment_method": "string",
  "status": "string",
  "start_time": "datetime",
  "end_time": "datetime",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### Price Response
```json
{
  "vehicle_type": "string",
  "reservation_time": "string",
  "price": "integer"
}
```

## Error Responses
- 400: Bad Request - Invalid input
- 401: Unauthorized - Authentication required
- 404: Not Found - Resource not found
- 409: Conflict - Resource conflict
- 500: Internal Server Error
