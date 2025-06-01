package service

/*import (
	"fmt"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/setupintent"
	"log"
)

func GetOrCreateStripeCustomer(email string, name string, existingStripeCustomerID string) (*stripe.Customer, error) {
	if existingStripeCustomerID != "" {
		params := &stripe.CustomerParams{}
		params.Email = stripe.String(email)
		params.Name = stripe.String(name)
		cust, err := customer.Get(existingStripeCustomerID, params)
		if err == nil && cust != nil && !cust.Deleted {
			return cust, nil
		}
		if err != nil && !stripe.IsErrorCode(err, stripe.ErrorCodeResourceMissing) {
			log.Printf("Error al obtener cliente de Stripe %s: %v", existingStripeCustomerID, err)
			// Decide si retornar el error o intentar crear uno nuevo
		}
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("error creando cliente de Stripe: %w", err)
	}
	return cust, nil
}

func CreateAndConfirmPaymentIntent(amount int64, currency string, paymentMethodID string, stripeCustomerID string, description string) (*stripe.PaymentIntent, error) {
	// 1. (Opcional pero recomendado) Adjuntar el PaymentMethod al Customer para uso futuro o mejor registro
	// Esto también lo puedes hacer al crear el PaymentIntent
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(stripeCustomerID),
	}
	_, err := paymentmethod.Attach(paymentMethodID, attachParams)
	if err != nil {
		// Puedes decidir si este error es fatal o si continúas e intentas el pago de todas formas
		log.Printf("Advertencia: no se pudo adjuntar PaymentMethod %s al cliente %s: %v", paymentMethodID, stripeCustomerID, err)
	}

	// 2. Crear el PaymentIntent
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amount),    // ej. 1000 para 10.00 EUR
		Currency:           stripe.String(currency), // ej. "eur"
		Customer:           stripe.String(stripeCustomerID),
		PaymentMethod:      stripe.String(paymentMethodID),
		ConfirmationMethod: stripe.String(string(stripe.PaymentIntentConfirmationMethodAutomatic)), // Stripe intenta confirmar inmediatamente
		Confirm:            stripe.Bool(true),                                                      // Confirmar el PaymentIntent inmediatamente
		Description:        stripe.String(description),                                             // ej. "Reserva GreenPark #12345"
		// Para SCA (Strong Customer Authentication) en Europa, podrías necesitar 'off_session' y 'return_url'
		// si la confirmación no es inmediata o requiere un paso 3D Secure.
		// Para cargos directos donde el usuario está presente (on-session), 'Automatic' y 'Confirm:true' suele ser el inicio.
		// Stripe maneja 3D Secure automáticamente si es necesario con esta configuración.
		// Si la tarjeta requiere autenticación 3D Secure, el PaymentIntent tendrá un estado como 'requires_action'
		// y el 'NextAction' contendrá la URL para la autenticación. El frontend deberá manejar esto.
		// Alternativamente, puedes hacer un PaymentIntent de dos pasos: crearlo y luego confirmarlo en el frontend.
		// Pero para un cargo directo simple, este flujo es común.
		ErrorOnRequiresAction: stripe.Bool(true), // Devuelve un error si se requiere acción del usuario (ej. 3DS)
		// en lugar de retornar un PaymentIntent en estado 'requires_action'.
		// El frontend puede necesitar manejar esto con confirmCardPayment.
		// Para simplificar el backend inicialmente, podemos empezar así y ver
		// si el frontend maneja la redirección de 3DS basándose en la respuesta.
		// Si usas `ErrorOnRequiresAction: stripe.Bool(true)`, el error devuelto si se requiere 3DS
		// será de tipo `*stripe.Error` y el `Code` será `stripe.ErrorCodePaymentIntentRequiresAction`.
	}
	// Podrías querer establecer 'SetupFutureUsage' si quieres guardar la tarjeta para después
	// params.SetupFutureUsage = stripe.String(string(stripe.PaymentIntentSetupFutureUsageOnSession))

	pi, err := paymentintent.New(params)
	if err != nil {
		// Manejar errores específicos de Stripe
		if stripeErr, ok := err.(*stripe.Error); ok {
			if stripeErr.Code == stripe.ErrorCodePaymentIntentRequiresAction {
				// Este es un caso especial: se necesita 3D Secure.
				// El frontend necesita manejar esto. El `pi` devuelto (incluso con error) tendrá el `NextAction`.
				log.Printf("PaymentIntent %s requiere acción del usuario (ej. 3DS). Client Secret: %s", pi.ID, pi.ClientSecret)
				// Devuelves el pi y un error especial o una forma de indicar al frontend que maneje el 3DS.
				// Por ahora, lo trataremos como un error que el handler debe interpretar.
				return pi, fmt.Errorf("pago requiere acción adicional del usuario (3DS): %s", pi.ID)
			}
		}
		return nil, fmt.Errorf("error creando PaymentIntent: %w", err)
	}

	// El PaymentIntent se creó y se intentó confirmar.
	// Verifica pi.Status: "succeeded", "requires_payment_method", "requires_confirmation", "requires_action", "processing", "canceled".
	// Si usaste Confirm:true y ConfirmationMethod:Automatic, esperas "succeeded" o "requires_action" (o un error).
	log.Printf("PaymentIntent %s creado con estado: %s", pi.ID, pi.Status)
	return pi, nil
}

func SetupCardForFutureUse(paymentMethodID string, stripeCustomerID string) (*stripe.SetupIntent, error) {
	// 1. Adjuntar el PaymentMethod al Customer
	// Esto es importante para que quede guardado y asociado.
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(stripeCustomerID),
	}
	pm, err := paymentmethod.Attach(paymentMethodID, attachParams)
	if err != nil {
		return nil, fmt.Errorf("error adjuntando PaymentMethod %s al cliente %s: %w", paymentMethodID, stripeCustomerID, err)
	}
	log.Printf("PaymentMethod %s adjuntado al cliente %s", pm.ID, stripeCustomerID)

	// 2. Crear y confirmar el SetupIntent para guardar la tarjeta para uso futuro (off-session)
	params := &stripe.SetupIntentParams{
		Customer:      stripe.String(stripeCustomerID),
		PaymentMethod: stripe.String(pm.ID),                                     // Usar el ID del PaymentMethod adjuntado
		Usage:         stripe.String(string(stripe.SetupIntentUsageOffSession)), // Indicar que se usará off-session
		Confirm:       stripe.Bool(true),                                        // Confirmar el SetupIntent ahora
		// Si se requiere autenticación (ej. 3DS para validar la tarjeta antes de guardarla),
		// Stripe lo manejará. El frontend podría necesitar intervenir si se devuelve 'requires_action'.
		// Similar a PaymentIntents, puedes usar ErrorOnRequiresAction.
		ErrorOnRequiresAction: stripe.Bool(true),
	}
	si, err := setupintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			if stripeErr.Code == stripe.ErrorCodeSetupIntentRequiresAction {
				log.Printf("SetupIntent %s requiere acción del usuario (ej. 3DS para validar tarjeta). Client Secret: %s", si.ID, si.ClientSecret)
				return si, fmt.Errorf("guardado de tarjeta requiere acción adicional del usuario (3DS): %s", si.ID)
			}
		}
		return nil, fmt.Errorf("error creando SetupIntent: %w", err)
	}

	// Verifica si.Status: "succeeded", "requires_payment_method", "requires_confirmation", "requires_action", "processing", "canceled".
	// Si Confirm:true, esperas "succeeded" o "requires_action" (o un error).
	// Si "succeeded", la tarjeta está lista para ser usada off-session con ese Customer.
	log.Printf("SetupIntent %s creado con estado: %s. Tarjeta guardada para cliente %s.", si.ID, si.Status, stripeCustomerID)

	// El PaymentMethod pm.ID es el que debes guardar en tu DB asociado a la reserva/usuario
	// para el cargo por no-show, o confiar en que el cliente de Stripe tendrá un payment_method por defecto o invoice_settings.
	// Stripe recomienda guardar el PaymentMethod ID que se usó en el SetupIntent.
	return si, nil
}*/
