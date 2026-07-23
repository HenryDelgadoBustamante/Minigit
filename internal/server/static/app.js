const app = {
  activeTab: 'dashboard',
  activeRepo: '',

  async init() {
    this.bindEvents();
    await this.loadProjects();
    this.refreshAll();
  },

  bindEvents() {
    // Navigation items
    document.querySelectorAll('.nav-item').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const tab = e.currentTarget.getAttribute('data-tab');
        this.switchTab(tab);
      });
    });

    // Refresh button
    document.getElementById('btn-refresh').addEventListener('click', () => {
      this.refreshAll();
    });

    // Repo Selector change
    const repoSel = document.getElementById('repo-selector');
    if (repoSel) {
      repoSel.addEventListener('change', (e) => {
        this.activeRepo = e.target.value;
        this.refreshAll();
      });
    }

    // Diff compare button
    document.getElementById('btn-compare').addEventListener('click', () => {
      const c1 = document.getElementById('diff-commit-1').value;
      const c2 = document.getElementById('diff-commit-2').value;
      this.loadDiff(c1, c2);
    });
  },

  async loadProjects() {
    try {
      const res = await fetch('/api/projects');
      const data = await res.json();

      const sel = document.getElementById('repo-selector');
      if (sel) {
        sel.innerHTML = '';
        (data.projects || []).forEach(p => {
          const opt = document.createElement('option');
          opt.value = p.name;
          opt.textContent = p.name;
          if (p.name === data.active) opt.selected = true;
          sel.appendChild(opt);
        });
      }
      this.activeRepo = data.active || '';
    } catch (err) {
      console.error('Error cargando lista de proyectos:', err);
    }
  },

  switchTab(tabName) {
    this.activeTab = tabName;

    document.querySelectorAll('.nav-item').forEach(btn => {
      btn.classList.toggle('active', btn.getAttribute('data-tab') === tabName);
    });

    document.querySelectorAll('.tab-content').forEach(sec => {
      sec.classList.toggle('active', sec.id === `view-${tabName}`);
    });

    const titles = {
      dashboard: ['Dashboard', 'Resumen general del estado del repositorio'],
      status: ['Estado del Repositorio', 'Archivos preparados, modificados y no rastreados'],
      files: ['Explorador de Archivos', 'Navega por la estructura de código en HEAD'],
      history: ['Historial de Commits', 'Línea de tiempo cronológica del proyecto'],
      'commit-detail': ['Detalle de Commit', 'Inspección de metadatos y cambios'],
      branches: ['Ramas del Repositorio', 'Punteros de desarrollo locales'],
      diff: ['Comparador Diff', 'Compara cambios entre dos versiones']
    };

    if (titles[tabName]) {
      document.getElementById('page-title').textContent = titles[tabName][0];
      document.getElementById('page-subtitle').textContent = titles[tabName][1];
    }

    if (tabName === 'status') this.loadStatus();
    if (tabName === 'files') this.loadFiles();
    if (tabName === 'history') this.loadHistory();
    if (tabName === 'branches') this.loadBranches();
    if (tabName === 'diff') this.loadDiffDropdowns();
  },

  async refreshAll() {
    await this.loadDashboard();
    if (this.activeTab === 'status') this.loadStatus();
    if (this.activeTab === 'files') this.loadFiles();
    if (this.activeTab === 'history') this.loadHistory();
    if (this.activeTab === 'branches') this.loadBranches();
  },

  async loadDashboard() {
    try {
      const res = await fetch(`/api/dashboard?repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      document.getElementById('sidebar-repo-path').textContent = data.repo_root || 'MiniGit Repo';
      document.getElementById('sidebar-repo-path').title = data.repo_root;
      document.getElementById('active-branch-name').textContent = data.active_branch;
      document.getElementById('stat-active-branch').textContent = data.active_branch;
      document.getElementById('stat-total-commits').textContent = data.total_commits;
      document.getElementById('stat-total-branches').textContent = data.total_branches;

      const pending = (data.staged_count || 0) + (data.unstaged_count || 0) + (data.untracked_count || 0);
      document.getElementById('stat-pending-changes').textContent = pending;

      const navPill = document.getElementById('nav-status-count');
      if (pending > 0) {
        navPill.textContent = pending;
        navPill.style.display = 'inline-block';
      } else {
        navPill.style.display = 'none';
      }

      document.getElementById('dash-staged-count').textContent = data.staged_count || 0;
      document.getElementById('dash-unstaged-count').textContent = data.unstaged_count || 0;
      document.getElementById('dash-untracked-count').textContent = data.untracked_count || 0;

      if (data.latest_commit) {
        document.getElementById('dash-commit-hash').textContent = data.latest_commit.hash.substring(0, 7);
        document.getElementById('dash-commit-msg').textContent = data.latest_commit.message;
        document.getElementById('dash-commit-author').textContent = `Autor: ${data.latest_commit.author}`;
        document.getElementById('dash-commit-date').textContent = new Date(data.latest_commit.timestamp).toLocaleString();
      } else {
        document.getElementById('dash-commit-msg').textContent = 'Sin commits registrados';
        document.getElementById('dash-commit-author').textContent = '';
        document.getElementById('dash-commit-date').textContent = '';
      }
    } catch (err) {
      console.error('Error cargando dashboard:', err);
    }
  },

  async loadStatus() {
    try {
      const res = await fetch(`/api/status?repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      this.renderStatusTable('table-staged', data.staged || []);
      this.renderStatusTable('table-unstaged', data.unstaged || []);
      this.renderUntrackedTable('table-untracked', data.untracked || []);
    } catch (err) {
      console.error('Error cargando status:', err);
    }
  },

  renderStatusTable(tableId, list) {
    const tbody = document.getElementById(tableId).querySelector('tbody');
    tbody.innerHTML = '';
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="2" class="text-muted" style="padding:1rem;">Sin archivos en esta categoría.</td></tr>';
      return;
    }

    list.forEach(item => {
      const tr = document.createElement('tr');
      let pillClass = 'pill-warning';
      let typeLabel = item.Type || 'modificado';

      if (typeLabel === 'added') { pillClass = 'pill-success'; typeLabel = 'Agregado (A)'; }
      else if (typeLabel === 'modified') { pillClass = 'pill-warning'; typeLabel = 'Modificado (M)'; }
      else if (typeLabel === 'deleted') { pillClass = 'pill-danger'; typeLabel = 'Eliminado (D)'; }

      tr.innerHTML = `
        <td><span class="pill ${pillClass}">${typeLabel}</span></td>
        <td><code>${item.Path}</code></td>
      `;
      tbody.appendChild(tr);
    });
  },

  renderUntrackedTable(tableId, list) {
    const tbody = document.getElementById(tableId).querySelector('tbody');
    tbody.innerHTML = '';
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="2" class="text-muted" style="padding:1rem;">Sin archivos no rastreados.</td></tr>';
      return;
    }

    list.forEach(path => {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td><span class="pill pill-danger">No Rastreado (?)</span></td>
        <td><code>${path}</code></td>
      `;
      tbody.appendChild(tr);
    });
  },

  async loadFiles() {
    try {
      const res = await fetch(`/api/tree?repo=${encodeURIComponent(this.activeRepo)}`);
      const entries = await res.json();

      const treeRoot = document.getElementById('file-tree-root');
      treeRoot.innerHTML = '';

      if (!entries || entries.length === 0) {
        treeRoot.innerHTML = '<li class="text-muted">Directorio vacío o sin commit</li>';
        return;
      }

      entries.forEach(entry => {
        const li = document.createElement('li');
        li.className = 'tree-item';
        const icon = entry.type === 'tree' ? '📁' : '📄';
        li.innerHTML = `<span>${icon}</span> <span>${entry.name}</span>`;

        if (entry.type === 'blob') {
          li.addEventListener('click', () => this.loadFileContent(entry.name, entry.hash));
        }
        treeRoot.appendChild(li);
      });
    } catch (err) {
      console.error('Error cargando lista de archivos:', err);
    }
  },

  async loadFileContent(fileName, hash) {
    try {
      document.getElementById('current-file-name').textContent = fileName;
      document.getElementById('current-file-hash').textContent = hash.substring(0, 7);

      const res = await fetch(`/api/file?hash=${hash}&repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      document.getElementById('file-content-code').textContent = data.content;
    } catch (err) {
      console.error('Error cargando archivo:', err);
    }
  },

  async loadHistory() {
    try {
      const res = await fetch(`/api/history?repo=${encodeURIComponent(this.activeRepo)}`);
      const commits = await res.json();

      const container = document.getElementById('timeline-container');
      container.innerHTML = '';

      if (!commits || commits.length === 0) {
        container.innerHTML = '<div class="placeholder-text">Sin historial de commits.</div>';
        return;
      }

      commits.forEach(c => {
        const item = document.createElement('div');
        item.className = 'timeline-item';
        item.innerHTML = `
          <div class="timeline-dot"></div>
          <div class="timeline-content">
            <h4>${c.Message}</h4>
            <div class="timeline-meta">
              <span class="mono-badge">${c.Hash.substring(0, 7)}</span> &bull;
              <span>${c.AuthorName} &lt;${c.AuthorEmail}&gt;</span> &bull;
              <span>${new Date(c.Timestamp).toLocaleString()}</span>
            </div>
          </div>
        `;
        item.addEventListener('click', () => this.openCommitDetail(c.Hash));
        container.appendChild(item);
      });
    } catch (err) {
      console.error('Error cargando historial:', err);
    }
  },

  async openCommitDetail(hash) {
    try {
      const res = await fetch(`/api/commit?hash=${hash}&repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      document.getElementById('cd-hash').textContent = data.hash;
      document.getElementById('cd-author').textContent = `${data.author_name} <${data.author_email}>`;
      document.getElementById('cd-date').textContent = new Date(data.timestamp).toLocaleString();
      document.getElementById('cd-tree').textContent = data.tree ? data.tree.substring(0, 7) : '---';
      document.getElementById('cd-parent').textContent = data.parent ? data.parent.substring(0, 7) : '(Inicial)';
      document.getElementById('cd-message').textContent = data.message;

      const changesList = document.getElementById('cd-changes-list');
      changesList.innerHTML = '';

      if (data.diff) {
        (data.diff.Added || []).forEach(p => {
          changesList.innerHTML += `<div style="color:var(--accent-green)">+ ${p} (Agregado)</div>`;
        });
        (data.diff.Modified || []).forEach(p => {
          changesList.innerHTML += `<div style="color:var(--accent-amber)">M ${p} (Modificado)</div>`;
        });
        (data.diff.Deleted || []).forEach(p => {
          changesList.innerHTML += `<div style="color:var(--accent-red)">- ${p} (Eliminado)</div>`;
        });
      }

      this.switchTab('commit-detail');
    } catch (err) {
      console.error('Error abriendo detalle de commit:', err);
    }
  },

  async loadBranches() {
    try {
      const res = await fetch(`/api/branches?repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      const tbody = document.getElementById('table-branches').querySelector('tbody');
      tbody.innerHTML = '';

      (data.branches || []).forEach(b => {
        const isActive = b.name === data.active;
        const tr = document.createElement('tr');
        tr.innerHTML = `
          <td><strong>${b.name}</strong> ${isActive ? '<span class="pill pill-success">Activa (*)</span>' : ''}</td>
          <td><code class="mono-badge">${b.commit ? b.commit.substring(0, 7) : '---'}</code></td>
          <td><code>minigit checkout ${b.name}</code></td>
        `;
        tbody.appendChild(tr);
      });
    } catch (err) {
      console.error('Error cargando ramas:', err);
    }
  },

  async loadDiffDropdowns() {
    try {
      const res = await fetch(`/api/history?repo=${encodeURIComponent(this.activeRepo)}`);
      const commits = await res.json();

      const select1 = document.getElementById('diff-commit-1');
      const select2 = document.getElementById('diff-commit-2');
      select1.innerHTML = '';
      select2.innerHTML = '';

      if (!commits || commits.length === 0) return;

      commits.forEach((c, idx) => {
        const opt1 = document.createElement('option');
        opt1.value = c.Hash;
        opt1.textContent = `${c.Hash.substring(0, 7)} - ${c.Message}`;

        const opt2 = opt1.cloneNode(true);

        select1.appendChild(opt1);
        select2.appendChild(opt2);
      });

      if (commits.length >= 2) {
        select1.selectedIndex = 1;
        select2.selectedIndex = 0;
      }
    } catch (err) {
      console.error('Error cargando dropdowns diff:', err);
    }
  },

  async loadDiff(c1, c2) {
    if (!c1 || !c2) return;
    try {
      const res = await fetch(`/api/diff?commit1=${c1}&commit2=${c2}&repo=${encodeURIComponent(this.activeRepo)}`);
      const data = await res.json();

      const container = document.getElementById('diff-results-container');
      container.innerHTML = '';

      if (!data.changes || data.changes.length === 0) {
        container.innerHTML = '<div class="placeholder-text">No existen diferencias entre los dos commits seleccionados.</div>';
        return;
      }

      data.changes.forEach(change => {
        const box = document.createElement('div');
        box.className = 'diff-box';

        let changeBadge = `<span class="pill pill-warning">${change.Type}</span>`;
        if (change.Type === 'A') changeBadge = `<span class="pill pill-success">Agregado (A)</span>`;
        if (change.Type === 'D') changeBadge = `<span class="pill pill-danger">Eliminado (D)</span>`;
        if (change.Type === 'R') changeBadge = `<span class="pill pill-warning">Renombrado (${change.OldPath} &rarr; ${change.Path})</span>`;

        box.innerHTML = `
          <div class="diff-header">${changeBadge} <span>${change.Path}</span></div>
          <div class="diff-content" id="diff-lines-${change.Path.replace(/[^a-zA-Z0-9]/g, '_')}">Cargando líneas...</div>
        `;
        container.appendChild(box);

        const elem = document.getElementById(`diff-lines-${change.Path.replace(/[^a-zA-Z0-9]/g, '_')}`);
        if (change.diff_lines && change.diff_lines.length > 0) {
          elem.innerHTML = change.diff_lines.map(line => {
            let cls = '';
            if (line.startsWith('+')) cls = 'diff-add';
            if (line.startsWith('-')) cls = 'diff-del';
            return `<div class="diff-line ${cls}">${this.escapeHtml(line)}</div>`;
          }).join('');
        } else {
          elem.innerHTML = '<div class="diff-line text-muted">Sin cambios de líneas de texto</div>';
        }
      });
    } catch (err) {
      console.error('Error calculando diff:', err);
    }
  },

  escapeHtml(str) {
    return str.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }
};

document.addEventListener('DOMContentLoaded', () => app.init());
