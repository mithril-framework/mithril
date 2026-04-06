(function () {
  var TOKEN_KEY = 'admin_token';

  var loginShell = document.getElementById('loginShell');
  var adminShell = document.getElementById('adminShell');
  var contentEl = document.getElementById('content');
  var sessionStatus = document.getElementById('sessionStatus');
  var welcomeName = document.getElementById('welcomeName');
  var bcRest = document.getElementById('bcRest');
  var toastEl = document.getElementById('toast');
  var modalOverlay = document.getElementById('modal-overlay');
  var modalTitle = document.getElementById('modal-title');
  var modalBody = document.getElementById('modal-body');
  var modalSave = document.getElementById('modal-save');

  var currentView = 'permissions';
  var modalKind = null;
  var reloadCurrent = null;

  var views = {
    permissions: { breadcrumb: 'Permissions', title: 'Select permission to change', addBtn: 'ADD PERMISSION +' },
    roles: { breadcrumb: 'Roles', title: 'Select role to change', addBtn: 'ADD ROLE +' },
    assign: { breadcrumb: 'Assign', title: 'Assign and revoke' },
    users: { breadcrumb: 'Users', title: 'Select user to change', addBtn: 'ADD USER +' },
    blogs: { breadcrumb: 'Blogs', title: 'Select blog to change', addBtn: 'ADD BLOG +' }
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
    } catch (e) {
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

  function updateBreadcrumb() {
    var v = views[currentView];
    if (!v) return;
    bcRest.innerHTML = '';
    var span = document.createElement('span');
    span.textContent = v.breadcrumb;
    bcRest.appendChild(span);
  }

  function setSidebarActive() {
    document.querySelectorAll('.sidebar-item .model-link').forEach(function (b) {
      b.classList.toggle('active', b.getAttribute('data-view') === currentView);
    });
  }

  function filterSidebar(val) {
    var v = (val || '').toLowerCase();
    document.querySelectorAll('.sidebar-item[data-label]').forEach(function (el) {
      var label = (el.getAttribute('data-label') || '').toLowerCase();
      el.style.display = !v || label.indexOf(v) >= 0 ? '' : 'none';
    });
  }

  function navigate(view) {
    if (!views[view]) return;
    currentView = view;
    setSidebarActive();
    updateBreadcrumb();
    render(view);
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
    setSidebarActive();
    updateBreadcrumb();
    navigate(currentView);
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

  function closeModal() {
    modalOverlay.classList.remove('open');
    modalKind = null;
    modalBody.innerHTML = '';
  }

  function openModal(kind) {
    modalKind = kind;
    modalBody.innerHTML = '';
    if (kind === 'permission') {
      modalTitle.textContent = 'Add permission';
      modalBody.innerHTML =
        '<div class="form-row"><label for="m_pcodename">Codename</label><input id="m_pcodename" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label for="m_pdesc">Description</label><input id="m_pdesc" type="text" autocomplete="off" /></div>';
    } else if (kind === 'role') {
      modalTitle.textContent = 'Add role';
      modalBody.innerHTML =
        '<div class="form-row"><label for="m_rname">Name</label><input id="m_rname" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label for="m_rdesc">Description</label><input id="m_rdesc" type="text" autocomplete="off" /></div>';
    } else if (kind === 'user') {
      modalTitle.textContent = 'Add user';
      modalBody.innerHTML =
        '<div class="form-row"><label for="m_uemail">Email</label><input id="m_uemail" type="email" autocomplete="off" /></div>' +
        '<div class="form-row"><label for="m_upass">Password</label><input id="m_upass" type="password" autocomplete="new-password" /></div>' +
        '<div class="form-row"><label for="m_ufirst">First name</label><input id="m_ufirst" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label for="m_ulast">Last name</label><input id="m_ulast" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label><input type="checkbox" id="m_uactive" checked /> Active</label></div>' +
        '<div class="form-row"><label><input type="checkbox" id="m_usuper" /> Superuser</label></div>';
    } else if (kind === 'blog') {
      modalTitle.textContent = 'Add blog';
      modalBody.innerHTML =
        '<div class="form-row"><label for="m_btitle">Title</label><input id="m_btitle" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label for="m_bcontent">Content</label><textarea id="m_bcontent" rows="5"></textarea></div>' +
        '<div class="form-row"><label for="m_bauthor">Author user id (UUID)</label><input id="m_bauthor" type="text" autocomplete="off" /></div>' +
        '<div class="form-row"><label><input type="checkbox" id="m_bactive" checked /> Active</label></div>';
    } else {
      modalKind = null;
      return;
    }
    modalOverlay.classList.add('open');
  }

  function saveModal() {
    if (!modalKind) return;
    var k = modalKind;
    var after = function () {
      closeModal();
      showToast('Saved.');
      if (reloadCurrent) reloadCurrent();
    };
    if (k === 'permission') {
      var codename = (document.getElementById('m_pcodename') || {}).value;
      var description = (document.getElementById('m_pdesc') || {}).value;
      api('/permissions', { method: 'POST', body: { codename: (codename || '').trim(), description: (description || '').trim() } })
        .then(after)
        .catch(errToast);
    } else if (k === 'role') {
      var name = (document.getElementById('m_rname') || {}).value;
      var rdesc = (document.getElementById('m_rdesc') || {}).value;
      api('/roles', { method: 'POST', body: { name: (name || '').trim(), description: (rdesc || '').trim() } })
        .then(after)
        .catch(errToast);
    } else if (k === 'user') {
      api('/resources/users', {
        method: 'POST',
        body: {
          email: (document.getElementById('m_uemail').value || '').trim(),
          password: document.getElementById('m_upass').value,
          first_name: (document.getElementById('m_ufirst').value || '').trim(),
          last_name: (document.getElementById('m_ulast').value || '').trim(),
          is_active: document.getElementById('m_uactive').checked,
          is_superuser: document.getElementById('m_usuper').checked
        }
      })
        .then(after)
        .catch(errToast);
    } else if (k === 'blog') {
      api('/resources/blogs', {
        method: 'POST',
        body: {
          title: document.getElementById('m_btitle').value,
          content: document.getElementById('m_bcontent').value,
          author_id: (document.getElementById('m_bauthor').value || '').trim(),
          is_active: document.getElementById('m_bactive').checked
        }
      })
        .then(after)
        .catch(errToast);
    }
  }

  function contentHeader(titleText, addBtnText, addKind) {
    var row = document.createElement('div');
    row.id = 'content-header';
    var h2 = document.createElement('h2');
    h2.textContent = titleText;
    row.appendChild(h2);
    if (addBtnText && addKind) {
      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'btn-add';
      btn.textContent = addBtnText;
      btn.addEventListener('click', function () {
        openModal(addKind);
      });
      row.appendChild(btn);
    }
    return row;
  }

  function renderPermissions() {
    contentEl.innerHTML = '';
    var meta = views.permissions;
    contentEl.appendChild(contentHeader(meta.title, meta.addBtn, 'permission'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    countEl.id = 'perm-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    var thead = document.createElement('thead');
    thead.innerHTML = '<tr><th>Codename</th><th>Description</th><th style="width:90px">Actions</th></tr>';
    table.appendChild(thead);
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
            td1.textContent = p.codename;
            var td2 = document.createElement('td');
            td2.textContent = p.description || '';
            var td3 = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/permissions/' + encodeURIComponent(p.codename), { method: 'DELETE' })
                .then(function () {
                  load();
                  showToast('Permission deleted.');
                })
                .catch(errToast);
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

  function renderRoles() {
    contentEl.innerHTML = '';
    var meta = views.roles;
    contentEl.appendChild(contentHeader(meta.title, meta.addBtn, 'role'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    var thead = document.createElement('thead');
    thead.innerHTML = '<tr><th>Name</th><th>Description</th><th style="width:90px">Actions</th></tr>';
    table.appendChild(thead);
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
            td1.textContent = r.name;
            var td2 = document.createElement('td');
            td2.textContent = r.description || '';
            var td3 = document.createElement('td');
            var del = document.createElement('button');
            del.type = 'button';
            del.className = 'btn-delete';
            del.textContent = 'Delete';
            del.addEventListener('click', function () {
              api('/roles/' + encodeURIComponent(r.name), { method: 'DELETE' })
                .then(function () {
                  load();
                  showToast('Role deleted.');
                })
                .catch(errToast);
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

  function renderAssign() {
    contentEl.innerHTML = '';
    contentEl.appendChild(contentHeader(views.assign.title, null, null));

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
    aUserEmail.id = 'aUserEmail';
    aUserEmail.placeholder = 'user email';
    var aRoleName = document.createElement('input');
    aRoleName.id = 'aRoleName';
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
    apRole.id = 'apRole';
    apRole.placeholder = 'role name';
    var apPerm = document.createElement('input');
    apPerm.id = 'apPerm';
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
    upUser.id = 'upUser';
    upUser.placeholder = 'user email';
    var upPerm = document.createElement('input');
    upPerm.id = 'upPerm';
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
      showToast('Done.');
    }
    function err(e) {
      setStatus(msg, e.message || String(e), 'err');
    }

    bAssign.addEventListener('click', function () {
      api('/assign/role', {
        method: 'POST',
        body: { user_email: aUserEmail.value.trim(), role_name: aRoleName.value.trim() }
      })
        .then(ok)
        .catch(err);
    });
    bRevoke.addEventListener('click', function () {
      api('/revoke/role', {
        method: 'POST',
        body: { user_email: aUserEmail.value.trim(), role_name: aRoleName.value.trim() }
      })
        .then(ok)
        .catch(err);
    });
    apA.addEventListener('click', function () {
      api('/assign/permission/role', {
        method: 'POST',
        body: { role_name: apRole.value.trim(), codename: apPerm.value.trim() }
      })
        .then(ok)
        .catch(err);
    });
    apR.addEventListener('click', function () {
      api('/revoke/permission/role', {
        method: 'POST',
        body: { role_name: apRole.value.trim(), codename: apPerm.value.trim() }
      })
        .then(ok)
        .catch(err);
    });
    upA.addEventListener('click', function () {
      api('/assign/permission/user', {
        method: 'POST',
        body: { user_email: upUser.value.trim(), codename: upPerm.value.trim() }
      })
        .then(ok)
        .catch(err);
    });
    upR.addEventListener('click', function () {
      api('/revoke/permission/user', {
        method: 'POST',
        body: { user_email: upUser.value.trim(), codename: upPerm.value.trim() }
      })
        .then(ok)
        .catch(err);
    });

    reloadCurrent = null;
  }

  function iconCell(yes) {
    var span = document.createElement('span');
    span.className = 'status-icon ' + (yes ? 'yes' : 'no');
    span.textContent = yes ? '✓' : '✕';
    return span;
  }

  function renderUsers() {
    contentEl.innerHTML = '';
    var meta = views.users;
    contentEl.appendChild(contentHeader(meta.title, meta.addBtn, 'user'));
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
            tdE.textContent = u.email;
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
              if (!window.confirm('Delete ' + u.email + '?')) return;
              api('/resources/users/' + u.id, { method: 'DELETE' })
                .then(function () {
                  load();
                  showToast('User deleted.');
                })
                .catch(errToast);
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

  function renderBlogs() {
    contentEl.innerHTML = '';
    var meta = views.blogs;
    contentEl.appendChild(contentHeader(meta.title, meta.addBtn, 'blog'));
    var countEl = document.createElement('div');
    countEl.className = 'result-count';
    contentEl.appendChild(countEl);
    var table = document.createElement('table');
    table.className = 'results-table';
    var thead = document.createElement('thead');
    thead.innerHTML = '<tr><th>Title</th><th>Id</th><th>Active</th><th style="width:90px">Actions</th></tr>';
    table.appendChild(thead);
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
            tdT.textContent = b.title || '(untitled)';
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
              if (!window.confirm('Delete this blog?')) return;
              api('/resources/blogs/' + b.id, { method: 'DELETE' })
                .then(function () {
                  load();
                  showToast('Blog deleted.');
                })
                .catch(errToast);
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

  function render(id) {
    if (id === 'permissions') renderPermissions();
    else if (id === 'roles') renderRoles();
    else if (id === 'assign') renderAssign();
    else if (id === 'users') renderUsers();
    else if (id === 'blogs') renderBlogs();
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
    navigate('permissions');
  });

  document.querySelectorAll('.sidebar-item .model-link').forEach(function (btn) {
    btn.addEventListener('click', function () {
      navigate(btn.getAttribute('data-view'));
    });
  });

  document.querySelectorAll('.sidebar-item .add-link').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var kind = btn.getAttribute('data-add');
      if (kind === 'permission' || kind === 'role' || kind === 'user' || kind === 'blog') openModal(kind);
    });
  });

  document.getElementById('sidebarFilter').addEventListener('input', function () {
    filterSidebar(this.value);
  });

  document.getElementById('modal-close').addEventListener('click', closeModal);
  document.getElementById('modal-cancel').addEventListener('click', closeModal);
  modalSave.addEventListener('click', saveModal);
  modalOverlay.addEventListener('click', function (e) {
    if (e.target === modalOverlay) closeModal();
  });

  verifySession();
})();
