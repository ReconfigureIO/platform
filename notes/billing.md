# Billing

We're going to have three tiers of plans

## Open source

id: "open-source"

Limited to 20 hours / month, we'll verify it somehow

## Single user

id: "single-user"

$250 for 80 hours / month
$4 / hour over

## Orgs

id: "organization"

$250/seat for 80 total hours across the org
$4 / hour over
Relies on Github's org model (not implemented yet)

# Payment info

We'll use Stripe for billing.

# Signup

Whenever a user signs up, they are restricted to open source. We'll
provide a path for them to include payment info in order to move to
the single user flow.

## Payment info flow

    * Stripe form for collecting payment info
    * Result token is POSTed to our servers
    * Our server creates a customer in Stripe, saves that info, and then signs them up for a plan stored in Stripe
    * We mark the user as having a plan

## API Changes

### GET /user

Web only?

```
curl -u $USER:$PASS -X GET localhost:8080/user
{"value":{"id":1,"name": "joshbohde", "email":"josh.bohde@reconfigure.io", "billing-plan": "open-source"}}
```
Can we share the stripe token, or should that be internal? If it's internal, we'll need to pull relevant info from Strip at that time

### POST /user/payment-info

```
curl -u $USER:$PASS -H "Content-Type: application/json" -X POST -d '{"stripe_token":"<strip_token>"}' http://localhost:8080/user/payment-info
```

### GET /user/invoices

Relevant info from the Stripe invoice objects
https://stripe.com/docs/api#invoice_object
