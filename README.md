axfrd
=====

This program provides a REST to AXFR functionality to be able to check if an
AXFR is possible for a given `zone` and `master IP` from the source IP provided
in the config file.

It solves the problem to make an AXFR request without being logged on the server
holding the source IP address.

[github.com/miekg/dns](https://github.com/miekg/dns) is used for the AXFR
requests. Any errors encountered by this go package during the zone transfer
will be part of the response to the REST request.

At the moment only the `axfr` endpoint is available. It accepts `POST` requests
with the following JSON data:

```
{
  "master": "172.19.254.13",
  "zone": "svc.1u1.it."
}
```

Example request:

```
curl -d '{"master":"199.4.138.53","zone":"example.com."}' 127.0.0.1:8080/axfr
```

The response contains the two fields `status` and `errormessage`. `status` can
have the value of `OK` or `Error`. If `status` contains `Error` the field
`errormessage` contains a string, otherwise it is empty.
