# URL Proxy

The URL proxy provides the single entrypoint for m3o.one

## Overview

The [url](https://github.com/micro/services) service provides link shortening and sharing. The URL Proxy fronts those urls 
as a single entrypoint at https://m3o.one. We don't serve directly because of time wasted on ssl certs, etc.
