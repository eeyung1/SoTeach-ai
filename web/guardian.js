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
  var accountPanel = document.getElementById('account-panel');
  var childrenList = document.getElementById('children-list');
  var plansList = document.getElementById('plans-list');

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
    accountPanel.hidden = false;
    msg('Signed in as ' + email(), '');
    loadAccount();
  }

  function showSignedOut() {
    consentPanel.hidden = true;
    accountPanel.hidden = true;
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
        loadAccount();
      } else {
        msg(res.data.error || ('Consent failed (' + res.status + ')'), 'error');
      }
    });
  });

  function fmtNaira(n) {
    try {
      return new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN', minimumFractionDigits: 0 }).format(n);
    } catch (e) {
      return 'NGN ' + n.toLocaleString();
    }
  }

  function authedRequest(method, path, body) {
    var headers = { 'Authorization': 'Bearer ' + token() };
    var init = { method: method, headers: headers };
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      init.body = JSON.stringify(body);
    }
    return fetch(path, init).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (data) {
        return { ok: res.ok, status: res.status, data: data };
      });
    });
  }

  function choosePlan(planID) {
    authedRequest('POST', '/guardians/plan', { planID: planID }).then(function (res) {
      if (res.ok) {
        msg('Plan selected: ' + res.data.plan.name + '. (Demo — no charge yet.)', '');
        loadAccount();
      } else if (res.status === 401) {
        signedOut();
        msg('Session expired. Please sign in again.', 'error');
      } else {
        msg(res.data.error || ('Could not select the plan (' + res.status + ')'), 'error');
      }
    }).catch(function () {
      msg('Could not reach the server. Is it running?', 'error');
    });
  }

  function renderChildren(list) {
    childrenList.innerHTML = '';
    if (!list || list.length === 0) {
      var none = document.createElement('li');
      none.className = 'muted';
      none.textContent = 'No children linked yet — give consent above.';
      childrenList.appendChild(none);
      return;
    }
    list.forEach(function (entry) {
      var li = document.createElement('li');
      li.textContent = entry.learner;
      childrenList.appendChild(li);
    });
  }

  function renderPlans(plans, activeID) {
    plansList.innerHTML = '';
    if (!plans || plans.length === 0) {
      return;
    }
    plans.forEach(function (plan) {
      var card = document.createElement('div');
      card.className = 'plan-card';
      var active = plan.id === activeID;
      if (active) {
        card.classList.add('active');
      }

      var info = document.createElement('div');
      info.className = 'plan-info';
      var name = document.createElement('h3');
      name.textContent = plan.name;
      var desc = document.createElement('p');
      desc.textContent = plan.description;
      info.appendChild(name);
      info.appendChild(desc);
      card.appendChild(info);

      var price = document.createElement('div');
      price.className = 'plan-price';
      price.textContent = fmtNaira(plan.priceNaira) + ' / ' + plan.period;
      card.appendChild(price);

      var btn = document.createElement('button');
      if (active) {
        btn.textContent = 'Active plan';
        btn.disabled = true;
        btn.className = 'btn btn-outline btn-sm';
      } else {
        btn.textContent = 'Choose this plan';
        btn.className = 'btn btn-primary btn-sm';
        btn.addEventListener('click', function () { choosePlan(plan.id); });
      }
      card.appendChild(btn);

      plansList.appendChild(card);
    });
  }

  function loadAccount() {
    if (!signedIn()) {
      return;
    }
    Promise.all([
      authedRequest('GET', '/guardians/me'),
      authedRequest('GET', '/guardians/plans'),
    ]).then(function (results) {
      var me = results[0];
      var plans = results[1];
      if (me.status === 401 || plans.status === 401) {
        signedOut();
        msg('Session expired. Please sign in again.', 'error');
        return;
      }
      if (!me.ok || !plans.ok) {
        msg('Could not load your account.', 'error');
        return;
      }
      var activeID = (me.data.plan && me.data.plan.plan && me.data.plan.plan.id) || '';
      renderChildren(me.data.learners);
      renderPlans(plans.data.plans, activeID);
    }).catch(function () {
      msg('Could not reach the server. Is it running?', 'error');
    });
  }

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
