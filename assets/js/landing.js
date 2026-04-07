// ─── PARSING DEMO STATE MACHINE ──────────────────────────────────────────────

const MSG = `Need ride now\nFrom Airport\nTo Downtown\nN$25 · 15km\n3 seats`;

const PARSED = {
  route: 'Airport → Downtown',
  departs: 'Now (immediate)',
  cost: 'N$25 · 15 km',
  seats: '3 available',
  status: 'Available',
};

const MATCHED = {
  driver: 'Amara K.  ★ 4.9',
  eta: '6 min',
  status: 'Confirmed · En route',
};

const chatText = document.getElementById('chatText');
const chatCursor = document.getElementById('chatCursor');
const scanLine = document.getElementById('scanLine');
const scanOverlay = document.getElementById('scanOverlay');
const rideDot = document.getElementById('rideDot');
const statusText = document.getElementById('demoStatusText');
const matchPanel = document.getElementById('matchPanel');
const matchPing = document.getElementById('matchPing');

const fields = [
  {
    wrap: document.getElementById('rf1'),
    val: document.getElementById('rfv1'),
    key: 'route',
  },
  {
    wrap: document.getElementById('rf2'),
    val: document.getElementById('rfv2'),
    key: 'departs',
  },
  {
    wrap: document.getElementById('rf3'),
    val: document.getElementById('rfv3'),
    key: 'cost',
  },
  {
    wrap: document.getElementById('rf4'),
    val: document.getElementById('rfv4'),
    key: 'seats',
  },
  {
    wrap: document.getElementById('rf5'),
    val: document.getElementById('rfv5'),
    key: 'status',
  },
];

const matchFields = [
  {
    wrap: document.getElementById('mf1'),
    val: document.getElementById('mfv1'),
    key: 'driver',
  },
  {
    wrap: document.getElementById('mf2'),
    val: document.getElementById('mfv2'),
    key: 'eta',
  },
  {
    wrap: document.getElementById('mf3'),
    val: document.getElementById('mfv3'),
    key: 'status',
  },
];

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function resetFields() {
  fields.forEach(f => {
    f.wrap.classList.remove('visible');
    f.val.textContent = '—';
  });
  rideDot.style.opacity = '0';
}

function resetMatch() {
  matchPanel.classList.remove('visible');
  matchPanel.style.opacity = '0';
  matchPanel.style.transform = 'translateY(6px)';
  matchFields.forEach(f => {
    f.wrap.classList.remove('visible');
    f.val.textContent = '—';
  });
  matchPing.style.opacity = '0';
}

async function typeText(text, el, speed = 35) {
  el.textContent = '';
  for (const ch of text) {
    el.textContent += ch;
    await sleep(ch === '\n' ? speed * 3 : speed);
  }
}

async function runScan() {
  scanLine.style.transition = 'none';
  scanLine.style.top = '0%';
  scanLine.style.opacity = '1';
  scanOverlay.style.opacity = '1';
  await sleep(50);
  scanLine.style.transition = 'top 1.4s linear';
  scanLine.style.top = '100%';
  await sleep(1600);
  scanLine.style.opacity = '0';
  scanOverlay.style.opacity = '0';
}

async function populateFields() {
  for (const f of fields) {
    f.wrap.classList.add('visible');
    await sleep(60);
    f.val.textContent = PARSED[f.key];
    await sleep(200);
  }
  rideDot.style.opacity = '1';
}

async function populateMatch() {
  matchPanel.style.transition = 'opacity 0.4s ease, transform 0.4s ease';
  matchPanel.style.opacity = '1';
  matchPanel.style.transform = 'none';
  await sleep(420);
  matchPing.style.opacity = '1';
  for (const f of matchFields) {
    f.wrap.classList.add('visible');
    await sleep(60);
    f.val.textContent = MATCHED[f.key];
    await sleep(240);
  }
}

async function demoLoop() {
  while (true) {
    // reset
    chatText.textContent = '';
    chatCursor.style.display = 'inline-block';
    resetFields();
    resetMatch();
    statusText.textContent = 'listening for messages…';
    await sleep(1200);

    // type message
    statusText.textContent = 'message received · parsing…';
    await typeText(MSG, chatText, 38);
    chatCursor.style.display = 'none';
    await sleep(600);

    // scan
    statusText.textContent = 'extracting structured data…';
    await runScan();
    await sleep(200);

    // populate ride card
    statusText.textContent = 'ride extracted ✓';
    await populateFields();
    await sleep(1000);

    // matching phase
    statusText.textContent = 'finding match…';
    await sleep(900);
    statusText.textContent = 'match found ✓  ·  ETA 6 min';
    await populateMatch();
    await sleep(2600);
  }
}

demoLoop();

// ─── SCROLL ANIMATIONS ────────────────────────────────────────────────────────

const fadeObserver = new IntersectionObserver(
  entries => {
    entries.forEach(e => {
      if (e.isIntersecting) {
        e.target.classList.add('visible');
      }
    });
  },
  { threshold: 0.15 }
);

document.querySelectorAll('.fade-in').forEach(el => {
  fadeObserver.observe(el);
});

// Phase bar progress animations
document.querySelectorAll('.phase-bar').forEach(bar => {
  const card = bar.closest('.phase-card');
  if (!card) return;

  const barObserver = new IntersectionObserver(
    entries => {
      entries.forEach(e => {
        if (e.isIntersecting) {
          setTimeout(() => {
            bar.style.width = `${bar.dataset.pct}%`;
          }, 200);
          barObserver.unobserve(e.target);
        }
      });
    },
    { threshold: 0.3 }
  );

  barObserver.observe(card);
});

// ─── SMOOTH NAV SCROLL ────────────────────────────────────────────────────────

document.querySelectorAll('a[href^="#"]').forEach(a => {
  a.addEventListener('click', e => {
    const id = a.getAttribute('href');
    if (id === '#') return;
    const target = document.querySelector(id);
    if (target) {
      e.preventDefault();
      target.scrollIntoView({ behavior: 'smooth' });
    }
  });
});
