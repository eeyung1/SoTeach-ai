/* Minimal guardian flow: register, log in (token kept in localStorage so a
   page reload keeps you signed in), and record consent for a learner. All
   rendered text is set via textContent, never innerHTML. */
(function () {
  'use strict';

  var TOKEN_KEY = 'soteach_guardian_token';
  var EMAIL_KEY = 'soteach_guardian_email';

  var authForms = document.getElementById('auth-forms');
  var registerForm = document.getElementById('register-form');
  var loginForm = document.getElementById('login-form');
  var consentPanel = document.getElementById('consent-panel');
  var consentForm = document.getElementById('consent-form');
  var learnerInput = document.getElementById('learner-name');
  var status = document.getElementById('status');

  function msg(text, kind) {
    status.hidden = false;
    status.textContent = text;
    status.style.color = (kind === 'error') ? '#b91c1c' : '';
  }

  function postJSON(path, body, token) {
    var headers = { 'Content-Type': 'application/json' };
    if (token) {
      headers.Authorization = 'Bearer ' + token;
    }
    return fetch(path, { method: 'POST', headers: headers, body: JSON.stringify(body) })
      .then(function (res) {
        return res.json().catch(function () { return {}; }).then(function (data) {
          return { ok: res.ok, status: res.status, data: data };
        });
      });
  }

  function token() {
    try { return localStorage.getItem(TOKEN_KEY) || ''; } catch (e) { return ''; }
  }

  function email() {
    try { return localStorage.getItem(EMAIL_KEY) || ''; } catch (e) { return ''; }
  }

  function signedIn() { return !!token(); }

  function showSignedIn() {
    authForms.hidden = true;
    consentPanel.hidden = false;
    msg('Signed in as ' + email(), '');
  }

  function showSignedOut() {
    consentPanel.hidden = true;
    authForms.hidden = false;
  }

  function signedOut() {
    try {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(EMAIL_KEY);
    } catch (e) { /* ignore */ }
    showSignedOut();
    msg('Signed out.', '');
  }

  function signedInAs(tok, em) {
    try {
      localStorage.setItem(TOKEN_KEY, tok);
      localStorage.setItem(EMAIL_KEY, em);
    } catch (e) { /* ignore */ }
    showSignedIn();
  }

  registerForm.addEventListener('submit', function (event) {
    event.preventDefault();
    postJSON('/guardians/register', {
      email: document.getElementById('reg-email').value.trim(),
      password: document.getElementById('reg-password').value,
    }).then(function (res) {
      if (res.ok) {
        msg('Account created — please log in.', '');
        loginForm.querySelector('button').focus();
      } else {
        msg(res.data.error || ('Register failed (' + res.status + ')'), 'error');
      }
    });
  });

  loginForm.addEventListener('submit', function (event) {
    event.preventDefault();
    postJSON('/guardians/login', {
      email: document.getElementById('login-email').value.trim(),
      password: document.getElementById('login-password').value,
    }).then(function (res) {
      if (res.ok) {
        signedInAs(res.data.token, res.data.email);
      } else {
        msg(res.data.error || ('Login failed (' + res.status + ')'), 'error');
      }
    });
  });

  consentForm.addEventListener('submit', function (event) {
    event.preventDefault();
    var learner = learnerInput.value.trim();
    if (!learner) { return; }
    postJSON('/guardians/consent', { learner: learner }, token()).then(function (res) {
      if (res.ok) {
        msg('Consent recorded for ' + learner + ' at ' + res.data.consentedAt + '.', '');
      } else {
        msg(res.data.error || ('Consent failed (' + res.status + ')'), 'error');
      }
    });
  });

  // A small "sign out" affordance in the status area.
  if (signedIn()) {
    showSignedIn();
    var out = document.createElement('button');
    out.textContent = 'Sign out';
    out.style.marginLeft = '0.75rem';
    out.addEventListener('click', signedOut);
    status.appendChild(out);
  } else {
    showSignedOut();
  }
})();
