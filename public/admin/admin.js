(function () {
  var TOKEN_KEY = 'admin_token';
  var api = function (path, opts) {
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
  };

  function setStatus(el, text, cls) {
    el.textContent = text || '';
    el.className = 'status' + (cls ? ' ' + cls : '');
  }

  document.getElementById('saveToken').onclick = function () {
    var v = document.getElementById('token').value.trim();
    sessionStorage.setItem(TOKEN_KEY, v);
    setStatus(document.getElementById('sessionStatus'), 'Token saved.', 'ok');
  };
  document.getElementById('clearToken').onclick = function () {
    sessionStorage.removeItem(TOKEN_KEY);
    document.getElementById('token').value = '';
    setStatus(document.getElementById('sessionStatus'), 'Cleared.');
  };

  var tabs = [
    { id: 'permissions', label: 'Permissions' },
    { id: 'roles', label: 'Roles' },
    { id: 'assign', label: 'Assign' },
    { id: 'users', label: 'Users' },
    { id: 'blogs', label: 'Blogs' }
  ];

  var tabBar = document.getElementById('tabs');
  var view = document.getElementById('view');
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

  function alertErr(e) {
    alert(e.message || String(e));
  }

  function render(id) {
    view.innerHTML = '';
    if (id === 'permissions') renderPermissions(view);
    else if (id === 'roles') renderRoles(view);
    else if (id === 'assign') renderAssign(view);
    else if (id === 'users') renderUsers(view);
    else if (id === 'blogs') renderBlogs(view);
  }

  render('permissions');
})();
