(function () {
  var TOKEN_KEY = 'admin_token';

  var loginShell = document.getElementById('loginShell');
  var adminShell = document.getElementById('adminShell');
  var contentEl = document.getElementById('content');
  var sessionStatus = document.getElementById('sessionStatus');
  var welcomeName = document.getElementById('welcomeName');
  var bcRest = document.getElementById('bcRest');
  var toastEl = document.getElementById('toast');

  var reloadCurrent = null;

  var labels = {
    permission: 'Permissions',
    role: 'Roles',
    user: 'Users',
    blog: 'Blogs',
    assign: 'Assign'
  };

  function setStatus(el, text, cls) {
    el.textContent = text || '';
    el.className = cls ? cls : '';
  }

  function showToast(msg) {
    toastEl.textContent = msg;
    toastEl.classList.add('show');
    setTimeout(function () { toastEl.classList.remove('show'); }, 3200);
  }

  function errToast(e) {
    showToast(e.message || String(e));
  }

  function parseJwtEmail(tok) {
    try {
      var parts = tok.split('.');
      if (parts.length < 2) return '';
      var b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
      var pad = b64.length % 4;
      if (pad) b64 += '===='.slice(0, 4 - pad);
      var payload = JSON.parse(atob(b64));
      return payload.email || payload.sub || '';
    } catch (err) {
      return '';
    }
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

  /** @returns {{ page: string, resource?: string, id?: string }} */
  function parseRoute() {
    var path = window.location.pathname.replace(/\/+$/, '') || '/admin';
    var rest = path.replace(/^\/admin\/?/, '').split('/').filter(Boolean);
    if (rest.length === 0) return { page: 'list', resource: 'permission' };
    var seg0 = rest[0];
    if (seg0 === 'assign') return { page: 'assign' };
    if (['permission', 'role', 'user', 'blog'].indexOf(seg0) < 0) {
      return { page: 'list', resource: 'permission' };
    }
    if (rest.length === 1) return { page: 'list', resource: seg0 };
    if (rest[1] === 'add') return { page: 'add', resource: seg0 };
    var id = rest.slice(1).map(function (s) { return decodeURIComponent(s); }).join('/');
    return { page: 'edit', resource: seg0, id: id };
  }

  function go(path) {
    if (path.indexOf('/') !== 0) path = '/admin/' + path;
    history.pushState(null, '', path);
    applyRoute();
  }

  function crumbLink(text, href) {
    var a = document.createElement('a');
    a.href = href;
    a.textContent = text;
    a.addEventListener('click', function (e) {
      e.preventDefault();
      go(href);
    });
    return a;
  }

  function buildBreadcrumbs(route) {
    bcRest.innerHTML = '';
    if (route.page === 'assign') {
      bcRest.appendChild(document.createTextNode(labels.assign));
      return;
    }
    var res = route.resource;
    var listHref = '/admin/' + res;
    bcRest.appendChild(crumbLink(labels[res] || res, listHref));
    if (route.page === 'add') {
      bcRest.appendChild(document.createTextNode(' › '));
      bcRest.appendChild(document.createTextNode('Add'));
    } else if (route.page === 'edit') {
      bcRest.appendChild(document.createTextNode(' › '));
      bcRest.appendChild(document.createTextNode('Change'));
    }
  }

  function setSidebarActive(route) {
    var base = '/admin/';
    if (route.page === 'assign') base = '/admin/assign';
    else if (route.resource) base = '/admin/' + route.resource;
    document.querySelectorAll('.sidebar-item .model-link').forEach(function (b) {
      var nav = b.getAttribute('data-nav');
      b.classList.toggle('active', nav === base);
    });
  }

  function filterSidebar(val) {
    var v = (val || '').toLowerCase();
    document.querySelectorAll('.sidebar-item[data-label]').forEach(function (el) {
      var label = (el.getAttribute('data-label') || '').toLowerCase();
      el.style.display = !v || label.indexOf(v) >= 0 ? '' : 'none';
    });
  }

  function applyRoute() {
    contentEl.innerHTML = '';
    reloadCurrent = null;
    var route = parseRoute();
    setSidebarActive(route);
    buildBreadcrumbs(route);

    if (route.page === 'assign') {
      renderAssign();
      return;
    }
    if (route.resource === 'permission') {
      if (route.page === 'list') renderPermissionList();
      else if (route.page === 'add') renderPermissionAdd();
      else renderPermissionView(route.id);
      return;
    }
    if (route.resource === 'role') {
      if (route.page === 'list') renderRoleList();
      else if (route.page === 'add') renderRoleAdd();
      else renderRoleView(route.id);
      return;
    }
    if (route.resource === 'user') {
      if (route.page === 'list') renderUserList();
      else if (route.page === 'add') renderUserAdd();
      else renderUserEdit(route.id);
      return;
    }
    if (route.resource === 'blog') {
      if (route.page === 'list') renderBlogList();
      else if (route.page === 'add') renderBlogAdd();
      else renderBlogEdit(route.id);
    }
  }

  function hideAdminUI() {
    adminShell.hidden = true;
    loginShell.hidden = false;
    contentEl.innerHTML = '';
    reloadCurrent = null;
  }

  function showAdminUI() {
    loginShell.hidden = true;
    adminShell.hidden = false;
    var tok = sessionStorage.getItem(TOKEN_KEY);
    var em = parseJwtEmail(tok || '');
    welcomeName.textContent = em ? em.toUpperCase() : 'ADMIN';
    applyRoute();
  }

  function verifySession() {
    var tok = sessionStorage.getItem(TOKEN_KEY);
    if (!tok) {
      hideAdminUI();
      setStatus(sessionStatus, 'Sign in with email/password or paste a token.', '');
      return Promise.resolve(false);
    }
    return fetch('/admin/api/meta', {
      headers: { Authorization: 'Bearer ' + tok, Accept: 'application/json' }
    })
      .then(function (res) {
        if (!res.ok) {
          sessionStorage.removeItem(TOKEN_KEY);
          document.getElementById('token').value = '';
          hideAdminUI();
          if (res.status === 401) setStatus(sessionStatus, 'Unauthorized — sign in again.', 'err');
          else if (res.status === 403) setStatus(sessionStatus, 'Forbidden — need superuser or admin.access permission.', 'err');
          else setStatus(sessionStatus, 'Could not verify session (' + res.status + ').', 'err');
          return false;
        }
        return res.json().then(function () {
          document.getElementById('token').value = tok;
          setStatus(sessionStatus, 'Signed in.', 'ok');
          showAdminUI();
          return true;
        });
      })
      .catch(function () {
        hideAdminUI();
        setStatus(sessionStatus, 'Network error — try again.', 'err');
        return false;
      });
  }

  function signOut() {
    sessionStorage.removeItem(TOKEN_KEY);
    document.getElementById('token').value = '';
    document.getElementById('loginEmail').value = '';
    document.getElementById('loginPassword').value = '';
    hideAdminUI();
    setStatus(sessionStatus, 'Signed out.', '');
  }

  function contentHeader(titleText, addLabel, addPath) {
    var row = document.createElement('div');
    row.id = 'content-header';
    var h2 = document.createElement('h2');
    h2.textContent = titleText;
    row.appendChild(h2);
    if (addLabel && addPath) {
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'btn-add';
      btn.textContent = addLabel;
      btn.addEventListener('click', function () { go(addPath); });
      row.appendChild(btn);
    }
    return row;
  }

  function pageFormShell(title, listPath) {
    var wrap = document.createElement('div');
    var header = document.createElement('div');
    header.id = 'content-header';
    var h2 = document.createElement('h2');
    h2.textContent = title;
    header.appendChild(h2);
    var back = document.createElement('button');
    back.type = 'button';
    back.className = 'btn-muted';
    back.textContent = '« Back to list';
    back.addEventListener('click', function () { go(listPath); });
    header.appendChild(back);
    wrap.appendChild(header);
    var err = document.createElement('p');
    err.className = 'modal-inline-err';
    err.id = 'page-form-err';
    wrap.appendChild(err);
    var panel = document.createElement('div');
    panel.className = 'assign-panel';
    wrap.appendChild(panel);
    return { wrap: wrap, panel: panel, err: err };
  }

  function attachPassMaskFocus(inp) {
    inp.readOnly = true;
    inp.addEventListener('focus', function once() {
      inp.readOnly = false;
      inp.removeEventListener('focus', once);
    });
  }

  function renderPermissionList() {
    contentEl.appendChild(contentHeader('Select permission to change', 'ADD PERMISSION +', '/admin/permission/add'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    table.innerHTML = '<thead><tr><th>Codename</th><th>Description</th><th style="width:120px">Actions</th></tr></thead>';
    var tbody = document.createElement('tbody');
    table.appendChild(tbody);
    contentEl.appendChild(table);

    function load() {
      api('/permissions')
        .then(function (rows) {
          tbody.innerHTML = '';
          countEl.innerHTML = '<strong>' + rows.length + '</strong> permission' + (rows.length === 1 ? '' : 's');
          rows.forEach(function (p) {
            var tr = document.createElement('tr');
            var td1 = document.createElement('td');
            var a = document.createElement('a');
            a.className = 'row-link';
            a.href = '/admin/permission/' + encodeURIComponent(p.codename);
            a.textContent = p.codename;
            a.addEventListener('click', function (e) {
              e.preventDefault();
              go('/admin/permission/' + encodeURIComponent(p.codename));
            });
            td1.appendChild(a);
            var td2 = document.createElement('td');
            td2.textContent = p.description || '';
            var td3 = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/permissions/' + encodeURIComponent(p.codename), { method: 'DELETE' }).then(load).catch(errToast);
            });
            td3.appendChild(del);
            tr.appendChild(td1);
            tr.appendChild(td2);
            tr.appendChild(td3);
            tbody.appendChild(tr);
          });
        })
        .catch(errToast);
    }
    reloadCurrent = load;
    load();
  }

  function renderPermissionAdd() {
    var shell = pageFormShell('Add permission', '/admin/permission');
    contentEl.appendChild(shell.wrap);
    shell.panel.innerHTML =
      '<div class="form-row"><label for="pa_codename">Codename</label><input id="pa_codename" type="text" autocomplete="off" /></div>' +
      '<div class="form-row"><label for="pa_desc">Description</label><input id="pa_desc" type="text" autocomplete="off" /></div>' +
      '<button type="button" class="btn-add" id="pa_save">Save</button>';
    document.getElementById('pa_save').addEventListener('click', function () {
      shell.err.textContent = '';
      var codename = (document.getElementById('pa_codename').value || '').trim();
      var description = (document.getElementById('pa_desc').value || '').trim();
      api('/permissions', { method: 'POST', body: { codename: codename, description: description } })
        .then(function () { go('/admin/permission'); })
        .catch(function (e) { shell.err.textContent = e.message || String(e); });
    });
  }

  function renderPermissionView(codename) {
    var shell = pageFormShell('Permission: ' + codename, '/admin/permission');
    contentEl.appendChild(shell.wrap);
    api('/permissions')
      .then(function (rows) {
        var p = rows.filter(function (x) { return x.codename === codename; })[0];
        if (!p) {
          shell.err.textContent = 'Permission not found.';
          return;
        }
        var hint = document.createElement('p');
        hint.className = 'signin-hint';
        hint.textContent = 'Codename cannot be changed via the API. Delete and recreate to rename.';
        shell.panel.appendChild(hint);
        var fr1 = document.createElement('div');
        fr1.className = 'form-row';
        var l1 = document.createElement('label');
        l1.textContent = 'Codename';
        var i1 = document.createElement('input');
        i1.type = 'text';
        i1.readOnly = true;
        i1.value = p.codename || '';
        fr1.appendChild(l1);
        fr1.appendChild(i1);
        shell.panel.appendChild(fr1);
        var fr2 = document.createElement('div');
        fr2.className = 'form-row';
        var l2 = document.createElement('label');
        l2.textContent = 'Description';
        var i2 = document.createElement('input');
        i2.type = 'text';
        i2.readOnly = true;
        i2.value = p.description || '';
        fr2.appendChild(l2);
        fr2.appendChild(i2);
        shell.panel.appendChild(fr2);
      })
      .catch(function (e) {
        shell.err.textContent = e.message || String(e);
      });
  }

  function renderRoleList() {
    contentEl.appendChild(contentHeader('Select role to change', 'ADD ROLE +', '/admin/role/add'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    table.innerHTML = '<thead><tr><th>Name</th><th>Description</th><th style="width:120px">Actions</th></tr></thead>';
    var tbody = document.createElement('tbody');
    table.appendChild(tbody);
    contentEl.appendChild(table);

    function load() {
      api('/roles')
        .then(function (rows) {
          tbody.innerHTML = '';
          countEl.innerHTML = '<strong>' + rows.length + '</strong> role' + (rows.length === 1 ? '' : 's');
          rows.forEach(function (r) {
            var tr = document.createElement('tr');
            var td1 = document.createElement('td');
            var a = document.createElement('a');
            a.className = 'row-link';
            a.href = '/admin/role/' + encodeURIComponent(r.name);
            a.textContent = r.name;
            a.addEventListener('click', function (e) {
              e.preventDefault();
              go('/admin/role/' + encodeURIComponent(r.name));
            });
            td1.appendChild(a);
            var td2 = document.createElement('td');
            td2.textContent = r.description || '';
            var td3 = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/roles/' + encodeURIComponent(r.name), { method: 'DELETE' }).then(load).catch(errToast);
            });
            td3.appendChild(del);
            tr.appendChild(td1);
            tr.appendChild(td2);
            tr.appendChild(td3);
            tbody.appendChild(tr);
          });
        })
        .catch(errToast);
    }
    reloadCurrent = load;
    load();
  }

  function renderRoleAdd() {
    var shell = pageFormShell('Add role', '/admin/role');
    contentEl.appendChild(shell.wrap);
    shell.panel.innerHTML =
      '<div class="form-row"><label for="ra_name">Name</label><input id="ra_name" type="text" autocomplete="off" /></div>' +
      '<div class="form-row"><label for="ra_desc">Description</label><input id="ra_desc" type="text" autocomplete="off" /></div>' +
      '<button type="button" class="btn-add" id="ra_save">Save</button>';
    document.getElementById('ra_save').addEventListener('click', function () {
      shell.err.textContent = '';
      api('/roles', {
        method: 'POST',
        body: {
          name: (document.getElementById('ra_name').value || '').trim(),
          description: (document.getElementById('ra_desc').value || '').trim()
        }
      })
        .then(function () { go('/admin/role'); })
        .catch(function (e) { shell.err.textContent = e.message || String(e); });
    });
  }

  function renderRoleView(name) {
    var shell = pageFormShell('Role: ' + name, '/admin/role');
    contentEl.appendChild(shell.wrap);
    api('/roles')
      .then(function (rows) {
        var r = rows.filter(function (x) { return x.name === name; })[0];
        if (!r) {
          shell.err.textContent = 'Role not found.';
          return;
        }
        var hint = document.createElement('p');
        hint.className = 'signin-hint';
        hint.textContent = 'Role name cannot be changed via the API. Delete and recreate to rename.';
        shell.panel.appendChild(hint);
        var fr1 = document.createElement('div');
        fr1.className = 'form-row';
        var l1 = document.createElement('label');
        l1.textContent = 'Name';
        var i1 = document.createElement('input');
        i1.type = 'text';
        i1.readOnly = true;
        i1.value = r.name || '';
        fr1.appendChild(l1);
        fr1.appendChild(i1);
        shell.panel.appendChild(fr1);
        var fr2 = document.createElement('div');
        fr2.className = 'form-row';
        var l2 = document.createElement('label');
        l2.textContent = 'Description';
        var i2 = document.createElement('input');
        i2.type = 'text';
        i2.readOnly = true;
        i2.value = r.description || '';
        fr2.appendChild(l2);
        fr2.appendChild(i2);
        shell.panel.appendChild(fr2);
      })
      .catch(function (e) {
        shell.err.textContent = e.message || String(e);
      });
  }

  function iconCell(yes) {
    var span = document.createElement('span');
    span.className = 'status-icon ' + (yes ? 'yes' : 'no');
    span.textContent = yes ? '✓' : '✕';
    return span;
  }

  function renderUserList() {
    contentEl.appendChild(contentHeader('Select user to change', 'ADD USER +', '/admin/user/add'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    var thead = document.createElement('thead');
    thead.innerHTML =
      '<tr><th>Email</th><th>Id</th><th>Superuser</th><th>Active</th><th style="width:90px">Actions</th></tr>';
    table.appendChild(thead);
    var tbody = document.createElement('tbody');
    table.appendChild(tbody);
    contentEl.appendChild(table);

    function load() {
      api('/resources/users?limit=50')
        .then(function (rows) {
          tbody.innerHTML = '';
          countEl.innerHTML = '<strong>' + rows.length + '</strong> user' + (rows.length === 1 ? '' : 's');
          rows.forEach(function (u) {
            var tr = document.createElement('tr');
            var tdE = document.createElement('td');
            var a = document.createElement('a');
            a.className = 'row-link';
            a.href = '/admin/user/' + u.id;
            a.textContent = u.email;
            a.addEventListener('click', function (e) {
              e.preventDefault();
              go('/admin/user/' + u.id);
            });
            tdE.appendChild(a);
            var tdI = document.createElement('td');
            tdI.textContent = u.id;
            var tdS = document.createElement('td');
            tdS.appendChild(iconCell(!!u.is_superuser));
            var tdA = document.createElement('td');
            tdA.appendChild(iconCell(u.is_active !== false));
            var tdX = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/resources/users/' + u.id, { method: 'DELETE' }).then(load).catch(errToast);
            });
            tdX.appendChild(del);
            tr.appendChild(tdE);
            tr.appendChild(tdI);
            tr.appendChild(tdS);
            tr.appendChild(tdA);
            tr.appendChild(tdX);
            tbody.appendChild(tr);
          });
        })
        .catch(errToast);
    }
    reloadCurrent = load;
    load();
  }

  function renderUserAdd() {
    var shell = pageFormShell('Add user', '/admin/user');
    contentEl.appendChild(shell.wrap);
    shell.panel.innerHTML =
      '<div class="form-row"><label for="ua_email">Email</label><input id="ua_email" type="text" inputmode="email" spellcheck="false" autocomplete="off" /></div>' +
      '<div class="form-row"><label for="ua_pass">Password</label><input id="ua_pass" type="text" class="admin-pass-mask" autocomplete="off" aria-label="Password" spellcheck="false" /></div>' +
      '<div class="form-row"><label for="ua_first">First name</label><input id="ua_first" type="text" autocomplete="off" /></div>' +
      '<div class="form-row"><label for="ua_last">Last name</label><input id="ua_last" type="text" autocomplete="off" /></div>' +
      '<div class="form-row"><label><input type="checkbox" id="ua_active" checked /> Active</label></div>' +
      '<div class="form-row"><label><input type="checkbox" id="ua_super" /> Superuser</label></div>' +
      '<button type="button" class="btn-add" id="ua_save">Save</button>';
    attachPassMaskFocus(document.getElementById('ua_pass'));
    document.getElementById('ua_save').addEventListener('click', function () {
      shell.err.textContent = '';
      var newEmail = (document.getElementById('ua_email').value || '').trim();
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)) {
        shell.err.textContent = 'Enter a valid email address.';
        return;
      }
      api('/resources/users', {
        method: 'POST',
        body: {
          email: newEmail,
          password: document.getElementById('ua_pass').value,
          first_name: (document.getElementById('ua_first').value || '').trim(),
          last_name: (document.getElementById('ua_last').value || '').trim(),
          is_active: document.getElementById('ua_active').checked,
          is_superuser: document.getElementById('ua_super').checked
        }
      })
        .then(function () { go('/admin/user'); })
        .catch(function (e) { shell.err.textContent = e.message || String(e); });
    });
  }

  function renderUserEdit(id) {
    var shell = pageFormShell('Change user', '/admin/user');
    contentEl.appendChild(shell.wrap);
    api('/resources/users/' + encodeURIComponent(id))
      .then(function (u) {
        shell.panel.innerHTML =
          '<div class="form-row"><label for="ue_email">Email</label><input id="ue_email" type="text" inputmode="email" autocomplete="off" /></div>' +
          '<div class="form-row"><label for="ue_pass">New password (leave blank to keep)</label>' +
          '<input id="ue_pass" type="text" class="admin-pass-mask" autocomplete="off" aria-label="New password" spellcheck="false" /></div>' +
          '<div class="form-row"><label for="ue_first">First name</label><input id="ue_first" type="text" autocomplete="off" /></div>' +
          '<div class="form-row"><label for="ue_last">Last name</label><input id="ue_last" type="text" autocomplete="off" /></div>' +
          '<div class="form-row"><label><input type="checkbox" id="ue_active" /> Active</label></div>' +
          '<div class="form-row"><label><input type="checkbox" id="ue_super" /> Superuser</label></div>' +
          '<button type="button" class="btn-add" id="ue_save">Save</button>';
        document.getElementById('ue_email').value = u.email || '';
        document.getElementById('ue_first').value = u.first_name || '';
        document.getElementById('ue_last').value = u.last_name || '';
        document.getElementById('ue_active').checked = u.is_active !== false;
        document.getElementById('ue_super').checked = !!u.is_superuser;
        attachPassMaskFocus(document.getElementById('ue_pass'));
        document.getElementById('ue_save').addEventListener('click', function () {
          shell.err.textContent = '';
          var newEmail = (document.getElementById('ue_email').value || '').trim();
          if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)) {
            shell.err.textContent = 'Enter a valid email address.';
            return;
          }
          var body = {
            email: newEmail,
            first_name: (document.getElementById('ue_first').value || '').trim(),
            last_name: (document.getElementById('ue_last').value || '').trim(),
            is_active: document.getElementById('ue_active').checked,
            is_superuser: document.getElementById('ue_super').checked
          };
          var np = document.getElementById('ue_pass').value;
          if (np) body.password = np;
          api('/resources/users/' + encodeURIComponent(id), { method: 'PUT', body: body })
            .then(function () { go('/admin/user'); })
            .catch(function (e) { shell.err.textContent = e.message || String(e); });
        });
      })
      .catch(function (e) {
        shell.err.textContent = e.message || String(e);
      });
  }

  function renderBlogList() {
    contentEl.appendChild(contentHeader('Select blog to change', 'ADD BLOG +', '/admin/blog/add'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    table.innerHTML = '<thead><tr><th>Title</th><th>Id</th><th>Active</th><th style="width:90px">Actions</th></tr></thead>';
    var tbody = document.createElement('tbody');
    table.appendChild(tbody);
    contentEl.appendChild(table);

    function load() {
      api('/resources/blogs?limit=50')
        .then(function (rows) {
          tbody.innerHTML = '';
          countEl.innerHTML = '<strong>' + rows.length + '</strong> blog' + (rows.length === 1 ? '' : 's');
          rows.forEach(function (b) {
            var tr = document.createElement('tr');
            var tdT = document.createElement('td');
            var a = document.createElement('a');
            a.className = 'row-link';
            a.href = '/admin/blog/' + b.id;
            a.textContent = b.title || '(untitled)';
            a.addEventListener('click', function (e) {
              e.preventDefault();
              go('/admin/blog/' + b.id);
            });
            tdT.appendChild(a);
            var tdI = document.createElement('td');
            tdI.textContent = b.id;
            var tdA = document.createElement('td');
            tdA.appendChild(iconCell(b.is_active !== false));
            var tdX = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/resources/blogs/' + b.id, { method: 'DELETE' }).then(load).catch(errToast);
            });
            tdX.appendChild(del);
            tr.appendChild(tdT);
            tr.appendChild(tdI);
            tr.appendChild(tdA);
            tr.appendChild(tdX);
            tbody.appendChild(tr);
          });
        })
        .catch(errToast);
    }
    reloadCurrent = load;
    load();
  }

  function renderBlogAdd() {
    var shell = pageFormShell('Add blog', '/admin/blog');
    contentEl.appendChild(shell.wrap);
    shell.panel.innerHTML =
      '<p class="signin-hint">Author is set automatically to your signed-in account.</p>' +
      '<div class="form-row"><label for="ba_title">Title</label><input id="ba_title" type="text" autocomplete="off" /></div>' +
      '<div class="form-row"><label for="ba_content">Content</label><textarea id="ba_content" rows="6"></textarea></div>' +
      '<div class="form-row"><label><input type="checkbox" id="ba_active" checked /> Active</label></div>' +
      '<button type="button" class="btn-add" id="ba_save">Save</button>';
    document.getElementById('ba_save').addEventListener('click', function () {
      shell.err.textContent = '';
      api('/resources/blogs', {
        method: 'POST',
        body: {
          title: document.getElementById('ba_title').value,
          content: document.getElementById('ba_content').value,
          is_active: document.getElementById('ba_active').checked
        }
      })
        .then(function () { go('/admin/blog'); })
        .catch(function (e) { shell.err.textContent = e.message || String(e); });
    });
  }

  function renderBlogEdit(id) {
    var shell = pageFormShell('Change blog', '/admin/blog');
    contentEl.appendChild(shell.wrap);
    api('/resources/blogs/' + encodeURIComponent(id))
      .then(function (b) {
        shell.panel.innerHTML =
          '<p class="signin-hint">Author cannot be changed here; it stays the original creator.</p>' +
          '<div class="form-row"><label for="be_title">Title</label><input id="be_title" type="text" autocomplete="off" /></div>' +
          '<div class="form-row"><label for="be_content">Content</label><textarea id="be_content" rows="6"></textarea></div>' +
          '<div class="form-row"><label><input type="checkbox" id="be_active" /> Active</label></div>' +
          '<button type="button" class="btn-add" id="be_save">Save</button>';
        document.getElementById('be_title').value = b.title || '';
        document.getElementById('be_content').value = b.content || '';
        document.getElementById('be_active').checked = b.is_active !== false;
        document.getElementById('be_save').addEventListener('click', function () {
          shell.err.textContent = '';
          api('/resources/blogs/' + encodeURIComponent(id), {
            method: 'PUT',
            body: {
              title: document.getElementById('be_title').value,
              content: document.getElementById('be_content').value,
              is_active: document.getElementById('be_active').checked
            }
          })
            .then(function () { go('/admin/blog'); })
            .catch(function (e) { shell.err.textContent = e.message || String(e); });
        });
      })
      .catch(function (e) {
        shell.err.textContent = e.message || String(e);
      });
  }

  function renderAssign() {
    contentEl.appendChild(contentHeader('Assign and revoke', null, null));

    function panel(title) {
      var p = document.createElement('div');
      p.className = 'assign-panel';
      var h3 = document.createElement('h3');
      h3.textContent = title;
      p.appendChild(h3);
      return p;
    }
    function row() {
      var r = document.createElement('div');
      r.className = 'assign-row';
      return r;
    }

    var msg = document.createElement('p');
    msg.id = 'assignMsg';

    var p1 = panel('Role ↔ user');
    var r1 = row();
    var aUserEmail = document.createElement('input');
    aUserEmail.placeholder = 'user email';
    var aRoleName = document.createElement('input');
    aRoleName.placeholder = 'role name';
    var bAssign = document.createElement('button');
    bAssign.type = 'button';
    bAssign.textContent = 'Assign role';
    var bRevoke = document.createElement('button');
    bRevoke.type = 'button';
    bRevoke.className = 'revoke';
    bRevoke.textContent = 'Revoke role';
    r1.appendChild(aUserEmail);
    r1.appendChild(aRoleName);
    r1.appendChild(bAssign);
    r1.appendChild(bRevoke);
    p1.appendChild(r1);

    var p2 = panel('Permission ↔ role');
    var r2 = row();
    var apRole = document.createElement('input');
    apRole.placeholder = 'role name';
    var apPerm = document.createElement('input');
    apPerm.placeholder = 'permission codename';
    var apA = document.createElement('button');
    apA.type = 'button';
    apA.textContent = 'Assign';
    var apR = document.createElement('button');
    apR.type = 'button';
    apR.className = 'revoke';
    apR.textContent = 'Revoke';
    r2.appendChild(apRole);
    r2.appendChild(apPerm);
    r2.appendChild(apA);
    r2.appendChild(apR);
    p2.appendChild(r2);

    var p3 = panel('Permission ↔ user (direct)');
    var r3 = row();
    var upUser = document.createElement('input');
    upUser.placeholder = 'user email';
    var upPerm = document.createElement('input');
    upPerm.placeholder = 'permission codename';
    var upA = document.createElement('button');
    upA.type = 'button';
    upA.textContent = 'Assign';
    var upR = document.createElement('button');
    upR.type = 'button';
    upR.className = 'revoke';
    upR.textContent = 'Revoke';
    r3.appendChild(upUser);
    r3.appendChild(upPerm);
    r3.appendChild(upA);
    r3.appendChild(upR);
    p3.appendChild(r3);

    contentEl.appendChild(p1);
    contentEl.appendChild(p2);
    contentEl.appendChild(p3);
    contentEl.appendChild(msg);

    function ok() {
      setStatus(msg, 'Done.', 'ok');
    }
    function errFn(e) {
      setStatus(msg, e.message || String(e), 'err');
    }

    bAssign.addEventListener('click', function () {
      api('/assign/role', {
        method: 'POST',
        body: { user_email: aUserEmail.value.trim(), role_name: aRoleName.value.trim() }
      }).then(ok).catch(errFn);
    });
    bRevoke.addEventListener('click', function () {
      api('/revoke/role', {
        method: 'POST',
        body: { user_email: aUserEmail.value.trim(), role_name: aRoleName.value.trim() }
      }).then(ok).catch(errFn);
    });
    apA.addEventListener('click', function () {
      api('/assign/permission/role', {
        method: 'POST',
        body: { role_name: apRole.value.trim(), codename: apPerm.value.trim() }
      }).then(ok).catch(errFn);
    });
    apR.addEventListener('click', function () {
      api('/revoke/permission/role', {
        method: 'POST',
        body: { role_name: apRole.value.trim(), codename: apPerm.value.trim() }
      }).then(ok).catch(errFn);
    });
    upA.addEventListener('click', function () {
      api('/assign/permission/user', {
        method: 'POST',
        body: { user_email: upUser.value.trim(), codename: upPerm.value.trim() }
      }).then(ok).catch(errFn);
    });
    upR.addEventListener('click', function () {
      api('/revoke/permission/user', {
        method: 'POST',
        body: { user_email: upUser.value.trim(), codename: upPerm.value.trim() }
      }).then(ok).catch(errFn);
    });

    reloadCurrent = null;
  }

  document.getElementById('loginBtn').addEventListener('click', function () {
    var email = document.getElementById('loginEmail').value.trim();
    var password = document.getElementById('loginPassword').value;
    if (!email || !password) {
      setStatus(sessionStatus, 'Email and password required.', 'err');
      return;
    }
    setStatus(sessionStatus, 'Signing in…', '');
    fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ email: email, password: password })
    })
      .then(function (res) {
        return res.json().then(function (body) {
          if (!res.ok) {
            var m = (body && body.message) || body.error || res.statusText;
            throw new Error(typeof m === 'string' ? m : 'Login failed');
          }
          var at = body.data && body.data.tokens && body.data.tokens.access_token;
          if (!at) throw new Error('No access_token in response');
          sessionStorage.setItem(TOKEN_KEY, at);
          document.getElementById('loginPassword').value = '';
          return verifySession();
        });
      })
      .catch(function (e) {
        setStatus(sessionStatus, e.message || 'Login failed', 'err');
      });
  });

  document.getElementById('saveToken').addEventListener('click', function () {
    var v = document.getElementById('token').value.trim();
    if (!v) {
      setStatus(sessionStatus, 'Paste a token first.', 'err');
      return;
    }
    sessionStorage.setItem(TOKEN_KEY, v);
    verifySession();
  });

  document.getElementById('clearToken').addEventListener('click', signOut);
  document.getElementById('headerSignOut').addEventListener('click', signOut);

  document.getElementById('bcHome').addEventListener('click', function (e) {
    e.preventDefault();
    go('/admin/');
  });

  adminShell.addEventListener('click', function (e) {
    var t = e.target.closest('[data-nav]');
    if (!t || !adminShell.contains(t)) return;
    e.preventDefault();
    go(t.getAttribute('data-nav'));
  });

  document.getElementById('sidebarFilter').addEventListener('input', function () {
    filterSidebar(this.value);
  });

  window.addEventListener('popstate', function () {
    if (!adminShell.hidden) applyRoute();
  });

  verifySession();
})();
