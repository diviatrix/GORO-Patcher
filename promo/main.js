(async function () {
  const container = document.getElementById('changelog-list');
  const tmpl = document.getElementById('release-card-tmpl');
  try {
    const res = await fetch('https://api.github.com/repos/diviatrix/GORO-Patcher/releases');
    if (!res.ok) throw new Error(res.status);
    const releases = await res.json();

    if (!releases.length) {
      container.replaceChildren(createMessage('No releases found.'));
      return;
    }

    if (releases[0]?.tag_name) {
      const el = document.getElementById('latest-ver');
      if (el) el.textContent = releases[0].tag_name;
    }

    const frag = document.createDocumentFragment();
    for (const r of releases) {
      frag.appendChild(buildCard(tmpl, r));
    }
    container.replaceChildren(frag);
  } catch (e) {
    const msg = createMessage('Could not load releases. ' + (e.message || '') + ' ');
    const link = document.createElement('a');
    link.href = 'https://github.com/diviatrix/GORO-Patcher/releases';
    link.textContent = 'View on GitHub';
    msg.appendChild(link);
    container.replaceChildren(msg);
  }
})();

function createMessage(text) {
  const p = document.createElement('p');
  p.className = 'changelog-msg';
  p.textContent = text;
  return p;
}

function buildCard(tmpl, r) {
  const card = tmpl.content.cloneNode(true);
  const q = sel => card.querySelector(`[data-field="${sel}"]`);

  q('ver').textContent = r.tag_name || r.name;

  q('date').textContent = new Date(r.published_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'short', day: 'numeric'
  });

  q('body').innerHTML = sanitize(mdToHtml(r.body || 'No release notes.', r.tag_name || 'main'));

  q('link').href = r.html_url;

  return card;
}

const GH_BASE = 'https://github.com/diviatrix/GORO-Patcher/raw/';

function ghUrl(url, ref) {
  if (/^https?:\/\/|^\/\//.test(url)) return url;
  return GH_BASE + (ref || 'main') + '/' + url.replace(/^\//, '');
}

function sanitize(html) {
  return html
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/\son\w+="[^"]*"/gi, '')
    .replace(/\son\w+='[^']*'/gi, '');
}

function esc(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function inlineMd(text, ref) {
  const htmlSlots = [];
  let out = text.replace(/<[^>]+>/g, m => {
    let fixed = m;
    fixed = fixed.replace(/(src|href)="([^"]+)"/i, (_, attr, val) => attr + '="' + ghUrl(val, ref) + '"');
    htmlSlots.push(fixed);
    return '\x00' + (htmlSlots.length - 1) + '\x00';
  });
  out = esc(out);
  out = out.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, url) => '<img src="' + ghUrl(url, ref) + '" alt="' + alt + '" class="release-img">');
  out = out.replace(/\[(!<img[^>]*>)\]\(([^)]+)\)/g, (_, img, url) => '<a href="' + ghUrl(url, ref) + '" target="_blank" rel="noopener">' + img + '</a>');
  out = out.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  out = out.replace(/`([^`]+)`/g, '<code>$1</code>');
  out = out.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, text, url) => '<a href="' + ghUrl(url, ref) + '" target="_blank" rel="noopener">' + text + '</a>');
  out = out.replace(/\x00(\d+)\x00/g, (_, i) => htmlSlots[i]);
  return out;
}

function mdToHtml(src, ref) {
  const lines = src.split('\n');
  let html = '';
  let inList = false;

  for (const raw of lines) {
    const trimmed = raw.trim();

    if (/^<img\s/i.test(trimmed)) {
      if (inList) { html += '</ul>'; inList = false; }
      let tag = trimmed
        .replace(/(width|height)="[^"]*"/gi, '')
        .replace(/<img\s/i, '<img class="release-img" ')
        .replace(/src="([^"]+)"/i, (_, src) => 'src="' + ghUrl(src, ref) + '"');
      html += tag;
      continue;
    }

    if (/^<br\s*\/?>$/i.test(trimmed)) {
      if (inList) { html += '</ul>'; inList = false; }
      continue;
    }

    if (/^#{2,3}\s+/.test(trimmed)) {
      if (inList) { html += '</ul>'; inList = false; }
      html += '<h3>' + inlineMd(trimmed.replace(/^#{2,3}\s+/, ''), ref) + '</h3>';
      continue;
    }

    if (/^[-*]\s+/.test(trimmed)) {
      const indent = raw.match(/^(\s*)/)[1].length;
      const content = trimmed.replace(/^[-*]\s+/, '');
      if (indent >= 2 && inList) {
        html += '<li class="sub">' + inlineMd(content, ref) + '</li>';
      } else {
        if (!inList) { html += '<ul>'; inList = true; }
        html += '<li>' + inlineMd(content, ref) + '</li>';
      }
      continue;
    }

    if (/^---+$/.test(trimmed)) {
      if (inList) { html += '</ul>'; inList = false; }
      html += '<hr>';
      continue;
    }

    if (inList && !trimmed) {
      html += '</ul>';
      inList = false;
      continue;
    }

    if (trimmed) {
      if (inList) { html += '</ul>'; inList = false; }
      html += '<p>' + inlineMd(trimmed, ref) + '</p>';
    }
  }

  if (inList) html += '</ul>';
  return html;
}

document.querySelectorAll('.zoomable').forEach(img => {
  img.addEventListener('click', () => {
    const pop = img.cloneNode(true);
    pop.classList.add('zoomable-pop');
    pop.addEventListener('click', () => pop.remove());
    document.body.appendChild(pop);
  });
});

const io = new IntersectionObserver(entries => {
  entries.forEach(e => {
    if (e.isIntersecting) {
      e.target.classList.add('visible');
      io.unobserve(e.target);
    }
  });
}, { threshold: 0.08 });
document.querySelectorAll('.reveal').forEach(el => io.observe(el));

(function () {
  const details = document.querySelector('#changelog details.release');
  if (!details) return;
  let loaded = false;
  details.addEventListener('toggle', async function () {
    if (!details.open || loaded) return;
    loaded = true;
    const container = document.getElementById('changelog-content');
    try {
      const res = await fetch('https://raw.githubusercontent.com/diviatrix/GORO-Patcher/main/doc/CHANGELOG.md');
      if (!res.ok) throw new Error(res.status);
      const md = await res.text();
      const cleaned = md.replace(/^#\s+Changelog\s*\n+/, '');
      container.innerHTML = sanitize(mdToHtml(cleaned, 'main'));
    } catch {
      const p = document.createElement('p');
      p.className = 'changelog-msg';
      p.textContent = 'Could not load changelog.';
      const link = document.createElement('a');
      link.href = 'https://github.com/diviatrix/GORO-Patcher/blob/main/doc/CHANGELOG.md';
      link.textContent = 'View on GitHub';
      p.appendChild(link);
      container.replaceChildren(p);
    }
  });
})();
