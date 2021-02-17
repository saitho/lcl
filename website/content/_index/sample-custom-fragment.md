+++
fragment = "content"
#disabled = true
date = "2017-10-05"
weight = 100
#background = ""
+++

## What is lcl?

lcl is a free Localhost redirect service and provides a valid Let's Encrypt SSL certificate.
It will redirect any request it receives to localhost.

This is useful if you need a valid HTTPS certificate which localhost can't provide.

## How to use lcl?

Using lcl is simple: Just add the path to this domain and set the target port as subdomain.

Below you find some examples. Opening the lcl domain will redirect your request to the target domain.

---

* **lcl domain**: https://80.lcl.ovh/oauth/authenticate
* **Target domain**: http://localhost/oauth/authenticate

---

* **lcl domain**: https://3000.lcl.ovh
* **Target domain**: http://localhost:3000
