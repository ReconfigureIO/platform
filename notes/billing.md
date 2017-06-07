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

## Monthly plans
All plans are calculated to begin at 00:00 GMT of every month.

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
{"value":{"name": "joshbohde", "email":"josh.bohde@reconfigure.io", "billing-plan": "open-source"}}
```
Can we share the stripe token, or should that be internal? If it's internal, we'll need to pull relevant info from Strip at that time

### PUT /user

Web only?

```
curl -u $USER:$PASS -X PUT -d "{\"email\": \"test@example.com\"}" localhost:8080/user
{"value":{"name": "joshbohde", "email":"test@example.com", "billing-plan": "open-source"}}
```
Can we share the stripe token, or should that be internal? If it's internal, we'll need to pull relevant info from Strip at that time

### GET /user/payment-info

```
curl -u $USER:$PASS http://localhost:8080/user/payment-info
{"value":{"id":"card_1ALCc1IXAoii2NU5rXwOZ9dV","exp_month":12,"exp_year":2018,"fingerprint":"SLmtIDa2sqd2iFnb","funding":"credit","last4":"4242","brand":"Visa","currency":"","default_for_currency":false,"address_city":"","address_country":"","address_line1":"","address_line1_check":"","address_line2":"","address_state":"","address_zip":"94301","address_zip_check":"pass","country":"US","customer":{"id":"cus_AgZQTeZbnY6AE4","livemode":false,"sources":null,"created":0,"account_balance":0,"currency":"","default_source":null,"delinquent":false,"description":"","discount":null,"email":"","metadata":null,"subscriptions":null,"deleted":false,"shipping":null,"business_vat_id":""},"cvc_check":"pass","metadata":{},"name":"","recipient":null,"dynamic_last4":"","deleted":false,"three_d_secure":null,"tokenization_method":"","description":"","iin":"","issuer":""}}
```


### POST /user/payment-info

```
curl -u $USER:$PASS -H "Content-Type: application/json" -X POST -d '{"stripe_token":"<strip_token>"}' http://localhost:8080/user/payment-info
{"value":{"id":"card_1ALCc1IXAoii2NU5rXwOZ9dV","exp_month":12,"exp_year":2018,"fingerprint":"SLmtIDa2sqd2iFnb","funding":"credit","last4":"4242","brand":"Visa","currency":"","default_for_currency":false,"address_city":"","address_country":"","address_line1":"","address_line1_check":"","address_line2":"","address_state":"","address_zip":"94301","address_zip_check":"pass","country":"US","customer":{"id":"cus_AgZQTeZbnY6AE4","livemode":false,"sources":null,"created":0,"account_balance":0,"currency":"","default_source":null,"delinquent":false,"description":"","discount":null,"email":"","metadata":null,"subscriptions":null,"deleted":false,"shipping":null,"business_vat_id":""},"cvc_check":"pass","metadata":{},"name":"","recipient":null,"dynamic_last4":"","deleted":false,"three_d_secure":null,"tokenization_method":"","description":"","iin":"","issuer":""}}
```

### GET /user/invoices

Relevant info from the Stripe invoice objects
https://stripe.com/docs/api#invoice_object
