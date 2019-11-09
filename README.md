# SAML Wireguard

- [Docker Python Alpine](https://hub.docker.com/_/python) base
- [python3-saml](https://github.com/onelogin/python3-saml) by OneLogin
- [Wireguard](https://wiki.archlinux.org/index.php/WireGuard) tools
- A few lines `/bin/sh`

<b>DISCLAIMER</b> - DO NOT USE IN PRODUCTION.

# SAML setup

To enable SAML, you will need to set up an application on your identity
provider's side, as well as configure your side. This example works with
OneLogin as the identity provider.

- [How SAML works](https://developers.onelogin.com/saml)

## OneLogin

Adjust the [.env](./example.env) file with your OneLogin settings:

```
ONELOGIN_DOMAIN= # Your organization's domain - https://<DOMAIN>.onelogin.com
ONELOGIN_CONNECTOR_ID= # The connector ID, copied from the SSO tab of your app
```

You also need to add OneLogin's certificate as `cert.pem`:

- Copy the X.509 certificate from `https://<DOMAIN>.onelogin.com/certificates`
- Save it as `cert.pem`
- Remove any line breaks, so that you end up having one single line in `cert.pem`

[This page](https://developers.onelogin.com/saml/python) has some additional
information.

If your OneLogin instance is <b>located in the US</b> you will have to change
`app-eu` to `app-us` in [saml/settings.json](./saml/settings.json). You can also
adjust this file to support other identity providers than OneLogin. Have a look
at the [python3-saml](https://github.com/onelogin/python3-saml) documentation.

## SAML service provider

Generate a service provider (`sp`) certificate and key as follows:

```
openssl req -new \
            -x509 \
            -days 3652 \
            -nodes \
        -out ./saml/certs/sp.crt \
        -keyout ./saml/certs/sp.key
```

# Wireguard

Adjust the `WG_` Wireguard variables in [.env](./example.env) as needed.

- [docker-entrypoint.sh](./docker-entrypoint.sh) will create the server's
configuration file and keys
- [wireguard.sh](./wireguard.sh) will add new peers

The application will only generate a VPN config when the SAML attribute
`memberOf` contains an entry `VPN`, assuming you have set up a corresponding
mapping in OneLogin. You can adjust the `vpn_access` behaviour in
[index.py](./index.py).

#### This is not a Wireguard server!

This application only generates the files for a server. You could bind mount the
[wireguard](./wireguard) directory to the host running the server, periodically
`docker cp` out the directory, share it over a network, or get it to the
Wireguard server any other way.

# Usage

1. Go through the [SAML setup](#saml-setup) above
1. `./run.sh` to build and run the image
1. Visit http://localhost:5000

<b>NOTE:</b> The first build is slow, the `xmlsec` layer takes a while. See
[Building](#building).

## Persistent data

- There's a [wireguard](./wireguard) volume in the [Dockerfile](./Dockerfile),
for testing you can remove the `--rm` flag from [run.sh](./run.sh)
- The [saml/certs](./saml/certs) directory is copied into the image

You could also bind mount these directories.

## Building

We're using Python 3, with Flask and OneLogin's
[python3-saml](https://github.com/onelogin/python3-saml). For versions, see
[requirements.txt](./requirements.txt).

Compiling `xmlsec` on Alpine will take a long time, but we end up with an image
half the size compared to Debian slim. From source is necessary because `xmlsec`
packages are broken for Alpine `>= 3.8` - see
[here](https://gitlab.alpinelinux.org/alpine/aports/issues/9110) and
[here](https://github.com/IdentityPython/pysaml2/issues/533#issuecomment-442386985).

### Image size

[Dive](https://github.com/wagoodman/dive) reports:

```
[Image Details]────────────────────────────────────

Total Image size: 200 MB
Potential wasted space: 1.3 MB
Image efficiency score: 99 %
```

