# One URL

One proxy to rule them all

## Overview

The M3O One proxy is a single proxy for a variety of our custom domains including m3o.one (short urls) and m3o.app (app urls).

## Apps

Apps are given a unique id and subdomain using m3o.app e.g helloworld.m3o.app resolves to the app id helloworld.

## URL

The [url](https://github.com/micro/services) service provides link shortening and sharing. The URL Proxy fronts those urls 
as a single entrypoint at https://m3o.one. We don't serve directly because of time wasted on ssl certs, etc.

- Assumes url is of format `https://m3o.one/u/AArfeZE`
- Will call `https://api.m3o.com/url/proxy?shortURL=https://m3o.one/u/AArfeZE`
- URL service should return `destinationURL=https://foobar.com/example`
- Proxy will issue a 301 redirect

## Usage

To deploy the url service

Set the host prefix

```
micro config set micro.url.host_prefix https://m3o.one/u/
micro config set micro.app.custom_domain m3o.app
```

Deploy the url, app services

```
micro run github.com/micro/services/app
micro run github.com/micro/services/url
```

Deploy this proxy to DO on domain m3o.one
