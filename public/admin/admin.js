(function () {
  var TOKEN_KEY = 'admin_token';
  var tabsBuilt = false;

  var adminApp = document.getElementById('adminApp');
  var tabBar = document.getElementById('tabs');
  var view = document.getElementById('view');
  var sessionStatus = document.getElementById('sessionStatus');

  var tabs = [
    { id: 'permissions', label: 'Permissions' },
    { id: 'roles', label: 'Roles' },
    { id: 'assign', label: 'Assign' },
    { id: 'users', label: 'Users' },
    { id: 'blogs', label: 'Blogs' }
  ];

  function setStatus(el, text, cls) {
    el.textContent = text || '';
    el.className = 'status' + (cls ? ' ' + cls : '');
  }

  function api(path, opts) {
    opts = opts || {};
    var tok = sessionStorage.getItem(TOKEN_KEY);
    var headers = opts.headers || {};
    if (tok) headers['Authorization'] = 'Bearer ' + tok;
    headers['Content-Type'] = headers['Content-Type'] || 'application/json';
    return fetch('/admin/api' + path, {
      method: opts.method || 'GET',
      headers: headers,
      body: opts.body ? JSON.stringify(opts.body) : undefined
    }).then(function (res) {
      if (res.status === 204 || res.status === 205) {
        if (!res.ok) throw new Error(res.statusText);
        return null;
      }
      var ct = res.headers.get('content-type') || '';
      var p = ct.indexOf('application/json') >= 0 ? res.json() : res.text().then(function (t) { return { _text: t }; });
      return p.then(function (data) {
        if (!res.ok) {
          var msg = (data && data.error) || data.message || data._text || res.statusText;
          throw new Error(typeof msg === 'string' ? msg : JSON.stringify(msg));
        }
        return data;
      });
    });
  }

  function hideAdminUI() {
    adminApp.hidden = true;
    view.innerHTML = '';
    tabBar.innerHTML = '';
    tabsBuilt = false;
  }

  function buildTabs() {
    if (tabsBuilt) return;
    tabsBuilt = true;
    tabs.forEach(function (t, i) {
      var b = document.createElement('button');
      b.type = 'button';
      b.textContent = t.label;
      b.dataset.tab = t.id;
      if (i === 0) b.classList.add('on');
      b.onclick = function () {
        tabBar.querySelectorAll('button').forEach(function (x) { x.classList.remove('on'); });
        b.classList.add('on');
        render(t.id);
      };
      tabBar.appendChild(b);
    });
  }

  function showAdminUI() {
    adminApp.hidden = false;
    buildTabs();
    render('permissions');
  }

  /** Validates token with server; shows admin UI only on success. No alert(). */
  function verifySession() {
    var tok = sessionStorage.getItem(TOKEN_KEY);
    if (!tok) {
      hideAdminUI();
      setStatus(sessionStatus, 'Sign in with email/password or paste a token.', '');
      return Promise.resolve(false);
    }
    return fetch('/admin/api/meta', {
      headers: { 'Authorization': 'Bearer ' + tok, 'Accept': 'application/json' }
    }).then(function (res) {
      if (!res.ok) {
        sessionStorage.removeItem(TOKEN_KEY);
        document.getElementById('token').value = '';
        hideAdminUI();
        if (res.status === 401) {
          setStatus(sessionStatus, 'Unauthorized — sign in again.', 'err');
        } else if (res.status === 403) {
          setStatus(sessionStatus, 'Forbidden — need superuser or admin.access permission.', 'err');
        } else {
          setStatus(sessionStatus, 'Could not verify session (' + res.status + ').', 'err');
        }
        return false;
      }
      return res.json().then(function () {
        document.getElementById('token').value = tok;
        setStatus(sessionStatus, 'Signed in.', 'ok');
        showAdminUI();
        return true;
      });
    }).catch(function () {
      hideAdminUI();
      setStatus(sessionStatus, 'Network error — try again.', 'err');
      return false;
    });
  }

  document.getElementById('loginBtn').onclick = function () {
    var email = document.getElementById('loginEmail').value.trim();
    var password = document.getElementById('loginPassword').value;
    if (!email || !password) {
      setStatus(sessionStatus, 'Email and password required.', 'err');
      return;
    }
    setStatus(sessionStatus, 'Signing in…', '');
    fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
      body: JSON.stringify({ email: email, password: password })
    })
      .then(function (res) {
        return res.json().then(function (body) {
          if (!res.ok) {
            var m = (body && body.message) || body.error || res.statusText;
            throw new Error(typeof m === 'string' ? m : 'Login failed');
          }
          var at =
            body.data &&
            body.data.tokens &&
            body.data.tokens.access_token;
          if (!at) throw new Error('No access_token in response');
          sessionStorage.setItem(TOKEN_KEY, at);
          document.getElementById('loginPassword').value = '';
          return verifySession();
        });
      })
      .catch(function (e) {
        setStatus(sessionStatus, e.message || 'Login failed', 'err');
      });
  };

  document.getElementById('saveToken').onclick = function () {
    var v = document.getElementById('token').value.trim();
    if (!v) {
      setStatus(sessionStatus, 'Paste a token first.', 'err');
      return;
    }
    sessionStorage.setItem(TOKEN_KEY, v);
    verifySession();
  };

  document.getElementById('clearToken').onclick = function () {
    sessionStorage.removeItem(TOKEN_KEY);
    document.getElementById('token').value = '';
    document.getElementById('loginEmail').value = '';
    document.getElementById('loginPassword').value = '';
    hideAdminUI();
    setStatus(sessionStatus, 'Signed out.', '');
  };

  function alertErr(e) {
    alert(e.message || String(e));
  }

  function renderPermissions(root) {
    var tpl = document.getElementById('tpl-permissions');
    root.appendChild(tpl.content.cloneNode(true));
    var listEl = root.querySelector('#pList');
    function load() {
      api('/permissions').then(function (rows) {
        listEl.innerHTML = '';
        rows.forEach(function (p) {
          var li = document.createElement('li');
          li.textContent = p.codename + ' — ' + (p.description || '');
          var del = document.createElement('button');
          del.type = 'button';
          del.textContent = 'Delete';
          del.onclick = function () {
            api('/permissions/' + encodeURIComponent(p.codename), { method: 'DELETE' }).then(function () { load(); }).catch(alertErr);
          };
          li.appendChild(del);
          listEl.appendChild(li);
        });
      }).catch(alertErr);
    }
    root.querySelector('#pCreate').onclick = function () {
      var codename = root.querySelector('#pCodename').value.trim();
      var description = root.querySelector('#pDesc').value.trim();
      api('/permissions', { method: 'POST', body: { codename: codename, description: description } })
        .then(load)
        .catch(alertErr);
    };
    load();
  }

  function renderRoles(root) {
    var tpl = document.getElementById('tpl-roles');
    root.appendChild(tpl.content.cloneNode(true));
    var listEl = root.querySelector('#rList');
    function load() {
      api('/roles').then(function (rows) {
        listEl.innerHTML = '';
        rows.forEach(function (r) {
          var li = document.createElement('li');
          li.textContent = r.name + ' — ' + (r.description || '');
          var del = document.createElement('button');
          del.type = 'button';
          del.textContent = 'Delete';
          del.onclick = function () {
            api('/roles/' + encodeURIComponent(r.name), { method: 'DELETE' }).then(function () { load(); }).catch(alertErr);
          };
          li.appendChild(del);
          listEl.appendChild(li);
        });
      }).catch(alertErr);
    }
    root.querySelector('#rCreate').onclick = function () {
      var name = root.querySelector('#rName').value.trim();
      var description = root.querySelector('#rDesc').value.trim();
      api('/roles', { method: 'POST', body: { name: name, description: description } })
        .then(load)
        .catch(alertErr);
    };
    load();
  }

  function renderAssign(root) {
    var tpl = document.getElementById('tpl-assign');
    root.appendChild(tpl.content.cloneNode(true));
    var msg = root.querySelector('#assignMsg');
    function ok() { setStatus(msg, 'Done.', 'ok'); }
    function err(e) { setStatus(msg, e.message, 'err'); }
    root.querySelector('#aRoleAssign').onclick = function () {
      api('/assign/role', { method: 'POST', body: { user_email: root.querySelector('#aUserEmail').value.trim(), role_name: root.querySelector('#aRoleName').value.trim() } })
        .then(ok).catch(err);
    };
    root.querySelector('#aRoleRevoke').onclick = function () {
      api('/revoke/role', { method: 'POST', body: { user_email: root.querySelector('#aUserEmail').value.trim(), role_name: root.querySelector('#aRoleName').value.trim() } })
        .then(ok).catch(err);
    };
    root.querySelector('#apAssign').onclick = function () {
      api('/assign/permission/role', { method: 'POST', body: { role_name: root.querySelector('#apRole').value.trim(), codename: root.querySelector('#apPerm').value.trim() } })
        .then(ok).catch(err);
    };
    root.querySelector('#apRevoke').onclick = function () {
      api('/revoke/permission/role', { method: 'POST', body: { role_name: root.querySelector('#apRole').value.trim(), codename: root.querySelector('#apPerm').value.trim() } })
        .then(ok).catch(err);
    };
    root.querySelector('#upAssign').onclick = function () {
      api('/assign/permission/user', { method: 'POST', body: { user_email: root.querySelector('#upUser').value.trim(), codename: root.querySelector('#upPerm').value.trim() } })
        .then(ok).catch(err);
    };
    root.querySelector('#upRevoke').onclick = function () {
      api('/revoke/permission/user', { method: 'POST', body: { user_email: root.querySelector('#upUser').value.trim(), codename: root.querySelector('#upPerm').value.trim() } })
        .then(ok).catch(err);
    };
  }

  function renderUsers(root) {
    var tpl = document.getElementById('tpl-users');
    root.appendChild(tpl.content.cloneNode(true));
    var listEl = root.querySelector('#uList');
    function load() {
      api('/resources/users?limit=50').then(function (rows) {
        listEl.innerHTML = '';
        rows.forEach(function (u) {
          var li = document.createElement('li');
          li.textContent = u.email + ' (' + u.id + ')' + (u.is_superuser ? ' [su]' : '');
          var ed = document.createElement('button');
          ed.type = 'button';
          ed.textContent = 'Delete';
          ed.onclick = function () {
            if (!confirm('Delete ' + u.email + '?')) return;
            api('/resources/users/' + u.id, { method: 'DELETE' }).then(load).catch(alertErr);
          };
          li.appendChild(ed);
          listEl.appendChild(li);
        });
      }).catch(alertErr);
    }
    root.querySelector('#uReload').onclick = load;
    root.querySelector('#uCreate').onclick = function () {
      api('/resources/users', {
        method: 'POST',
        body: {
          email: root.querySelector('#uEmail').value.trim(),
          password: root.querySelector('#uPass').value,
          first_name: root.querySelector('#uFirst').value.trim(),
          last_name: root.querySelector('#uLast').value.trim(),
          is_active: root.querySelector('#uActive').checked,
          is_superuser: root.querySelector('#uSuper').checked
        }
      }).then(load).catch(alertErr);
    };
    load();
  }

  function renderBlogs(root) {
    var tpl = document.getElementById('tpl-blogs');
    root.appendChild(tpl.content.cloneNode(true));
    var listEl = root.querySelector('#bList');
    function load() {
      api('/resources/blogs?limit=50').then(function (rows) {
        listEl.innerHTML = '';
        rows.forEach(function (b) {
          var li = document.createElement('li');
          li.textContent = (b.title || '(untitled)') + ' — ' + b.id;
          var del = document.createElement('button');
          del.type = 'button';
          del.textContent = 'Delete';
          del.onclick = function () {
            if (!confirm('Delete blog?')) return;
            api('/resources/blogs/' + b.id, { method: 'DELETE' }).then(load).catch(alertErr);
          };
          li.appendChild(del);
          listEl.appendChild(li);
        });
      }).catch(alertErr);
    }
    root.querySelector('#bReload').onclick = load;
    root.querySelector('#bCreate').onclick = function () {
      api('/resources/blogs', {
        method: 'POST',
        body: {
          title: root.querySelector('#bTitle').value,
          content: root.querySelector('#bContent').value,
          author_id: root.querySelector('#bAuthor').value.trim(),
          is_active: root.querySelector('#bActive').checked
        }
      }).then(load).catch(alertErr);
    };
    load();
  }

  function render(id) {
    view.innerHTML = '';
    if (id === 'permissions') renderPermissions(view);
    else if (id === 'roles') renderRoles(view);
    else if (id === 'assign') renderAssign(view);
    else if (id === 'users') renderUsers(view);
    else if (id === 'blogs') renderBlogs(view);
  }

  verifySession();
})();
