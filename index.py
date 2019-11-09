import os

from flask import (Flask, request, render_template, redirect, session,
                   make_response)

# Needed for OneLogin / SAML
from urllib.parse import urlparse

from onelogin.saml2.auth import OneLogin_Saml2_Auth
from onelogin.saml2.utils import OneLogin_Saml2_Utils
from onelogin.saml2.errors import OneLogin_Saml2_Error

# Needed for generating Wireguard configuration files
from ipaddress import IPv4Network
import mmap
import subprocess
import sys

network = IPv4Network(os.getenv('WG_ADDRESS'))
hosts_iterator = (host for host in network.hosts())

app = Flask(__name__)
app.config['SECRET_KEY'] = 'onelogindemopytoolkit' # FIXME
app.config['SAML_PATH'] = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'saml')


def init_saml_auth(req):
    auth = OneLogin_Saml2_Auth(req, custom_base_path=app.config['SAML_PATH'])
    return auth


def prepare_flask_request(request):
    # If server is behind proxys or balancers use the HTTP_X_FORWARDED fields
    url_data = urlparse(request.url)
    return {
        'https': 'on' if request.scheme == 'https' else 'off',
        'http_host': request.host,
        'server_port': url_data.port,
        'script_name': request.path,
        'get_data': request.args.copy(),
        # Uncomment if using ADFS as IdP, https://github.com/onelogin/python-saml/pull/144
        # 'lowercase_urlencoding': True,
        'post_data': request.form.copy()
    }


def generate_wireguard_config(email):
    with open('./wireguard/wireguard.conf', 'rb', 0) as file, \
        mmap.mmap(file.fileno(), 0, access=mmap.ACCESS_READ) as s:
        search_for = "# Client: " + email
        if s.find(bytes(search_for, encoding='utf8')) != -1:
            print('Found Wireguard config for ' + email,
                  file=sys.stderr)
        else:
            ip = str(next(hosts_iterator))
            while s.find(bytes(ip, encoding='utf8')) != -1:
                ip = str(next(hosts_iterator))
            print('Not found, creating Wireguard config for ' + email,
                  file=sys.stderr)
            subprocess.run(["./wireguard.sh", email, ip])
    config_file = "./wireguard/" + email + '.conf'
    with open(config_file, 'r') as file:
        client_config = file.read()
    return client_config


@app.route('/', methods=['GET', 'POST'])
def index():
    req = prepare_flask_request(request)
    auth = init_saml_auth(req)
    errors = []
    not_auth_warn = False
    success_slo = False
    vpn_access = False
    vpn_config = '',
    attributes = True
    paint_logout = False

    if 'sso' in request.args:
        sso_built_url = auth.login()
        session['AuthNRequestID'] = auth.get_last_request_id()
        return redirect(sso_built_url)
    elif 'acs' in request.args:
        request_id = None
        if 'AuthNRequestID' in session:
            request_id = session['AuthNRequestID']

        try:
            auth.process_response(request_id=request_id)
        except OneLogin_Saml2_Error:
            errors = OneLogin_Saml2_Error
            pass

        errors = auth.get_errors()
        not_auth_warn = not auth.is_authenticated()

        if len(errors) == 0:
            if auth.is_authenticated():
                if 'AuthNRequestID' in session:
                    del session['AuthNRequestID']
                session['samlNameId'] = auth.get_nameid()
                session['samlNameIdFormat'] = auth.get_nameid_format()
                session['samlNameIdNameQualifier'] = auth.get_nameid_nq()
                session['samlNameIdSPNameQualifier'] = auth.get_nameid_spnq()
                session['samlSessionIndex'] = auth.get_session_index()
                session['samlUserdata'] = auth.get_attributes()
                self_url = OneLogin_Saml2_Utils.get_self_url(req)

    if 'samlUserdata' in session:
        if auth.is_authenticated():
            paint_logout = True
            if len(session['samlUserdata']) > 0:
                attributes = session['samlUserdata'].items()
                client_email = session['samlUserdata']['User.email'][0]

                if 'VPN' in session['samlUserdata']['memberOf']:
                    vpn_access = True
                    vpn_config = generate_wireguard_config(client_email)
                else:
                    vpn_config = 'Access denied.'
                    print('Access denied for ' + client_email,
                          file=sys.stderr)

    return render_template(
        'index.html',
        errors=errors,
        vpn_access=vpn_access,
        vpn_config=vpn_config,
        not_auth_warn=not_auth_warn,
        success_slo=success_slo,
        attributes=attributes,
        paint_logout=paint_logout
    )


@app.route('/metadata/')
def metadata():
    req = prepare_flask_request(request)
    auth = init_saml_auth(req)
    settings = auth.get_settings()
    metadata = settings.get_sp_metadata()
    errors = settings.validate_metadata(metadata)

    if len(errors) == 0:
        resp = make_response(metadata, 200)
        resp.headers['Content-Type'] = 'text/xml'
    else:
        resp = make_response(', '.join(errors), 500)
    return resp


if __name__ == "__main__":
    app.run(host='0.0.0.0', port=5000, debug=True)
