
const loadedSections = {};
let serverList = [];

function parseProxyUrl(url) {
	const urlObj = new URL(url);
	return {
		username: urlObj.username,
		password: urlObj.password,
		host: urlObj.hostname,
		port: urlObj.port
	};
}

async function decrypt(ks, ed) {
    const kb = new TextEncoder().encode(ks);
    if (kb.length !== 16) {
        throw new Error(`Key must be 16 bytes, got ${kb.length}`);
    }
    const binaryString = atob(ed);
    const encrypted = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        encrypted[i] = binaryString.charCodeAt(i);
    }
    const nonce = encrypted.slice(0, 12);
    const ciphertext = encrypted.slice(12);
    const ck = await crypto.subtle.importKey(
        'raw',
        kb,
        { name: 'AES-GCM' },
        false,
        ['decrypt']
    );
    const decrypted = await crypto.subtle.decrypt(
        {
            name: 'AES-GCM',
            iv: nonce
        },
        ck,
        ciphertext
    );
    return new TextDecoder().decode(decrypted);
}

function updateServerLoad(value) {
	const slider = document.getElementById('serverLoadSlider');
	const counter = document.getElementById('serverLoadSliderCounter');
	const numeric = Math.max(0, Math.min(100, Number(value) || 0));
	slider.style.left = Math.round(numeric) + '%';
	counter.textContent = numeric.toFixed(1) + '%';
}

function createCopyButton(textProvider) {
	const btn = document.createElement('button');
	btn.className = 'copyBtn'; btn.type = 'button';
	btn.textContent = '📋';
	btn.addEventListener('click', () => navigator.clipboard.writeText(textProvider()));
	return btn;
}

function appendProxyBlock(container, title, items, formatter = (v) => v) {
	if (!items || items.length === 0) return;
	const header = document.createElement('h3');
	header.innerText = title;
	container.appendChild(header);
	container.appendChild(document.createElement('hr'));
	items.forEach(item => {
		const row = document.createElement('p');
		row.className = 'flex-jc-sb';
		const code = document.createElement('code');
		const pre = document.createElement('pre');
		pre.innerText = formatter(item);
		code.appendChild(pre);
		row.appendChild(code);
		row.appendChild(createCopyButton(() => pre.innerText));
		container.appendChild(row);
	});
}

function renderSmartProxyBlock(container, httpList, serverName) {
	if (!httpList || httpList.length === 0) return;
	const header = document.createElement('h3');
	header.innerText = 'SmartProxy Servers';
	container.appendChild(header);
	container.appendChild(document.createElement('hr'));
	let content = '[SmartProxy Servers]\n';
	httpList.forEach(h => {
		try {
			const info = parseProxyUrl(h);
			content += `${info.host}:${info.port} [HTTP] [${serverName}_${info.username}] [${info.username}] [${info.password}]\n`;
		} catch (e) { }
	});
	const row = document.createElement('p');
	row.className = 'flex-jc-sb';
	const code = document.createElement('code');
	const pre = document.createElement('pre');
	pre.innerText = content;
	code.appendChild(pre);
	row.appendChild(code);
	row.appendChild(createCopyButton(() => pre.innerText));
	container.appendChild(row);
}

async function showServer(index) {
	document.querySelectorAll('#serversTable button').forEach(b => b.disabled = false);
	const buttons = document.querySelectorAll('#serversTable button');
	const targetBtn = Array.from(buttons).find(b => Number(b.dataset.index) === index);
	if (targetBtn) targetBtn.disabled = true;
	const nextBtn = document.getElementById('nextServerBtn');
	const nextIndex = (index + 1) % serverList.length;
	nextBtn.textContent = serverList.length > 0 ? `→ ${serverList[nextIndex].name}` : '';
	nextBtn.onclick = () => showServer(nextIndex);
	nextBtn.disabled = serverList.length <= 1;
	document.getElementById('serverContentName').textContent = serverList[index].name || '';

	const providerNameEl = document.getElementById('providerName');
	const tariffPlanEl = document.getElementById('tariffPlan');
	const speedLimitEl = document.getElementById('speedLimit');
	const trafficLimitEl = document.getElementById('trafficLimit');

	const providerLink = document.createElement('a');
	providerLink.href = serverList[index].providerLink || '#';
	providerLink.innerText = serverList[index].providerName || '';
	providerNameEl.innerHTML = '';
	providerNameEl.appendChild(providerLink);

	tariffPlanEl.textContent = serverList[index].plan || '';
	speedLimitEl.textContent = serverList[index].speedRate || '';
	trafficLimitEl.textContent = serverList[index].limit || '';

	document.getElementById('serverInfoTable').hidden = false;
	document.getElementById('serverLoadHeader').hidden = false;
	document.getElementById('serverLoadScaleContainer').hidden = false;
	document.getElementById('serverHeaderContainer').hidden = false;

	(async () => {
		try {
			const res = await fetch(`./serverinfo/?url=${serverList[index].infoLink}/stat`);
			if (!res.ok) throw new Error('stat fetch failed');
			const stat = await res.json();
			const day30Tx = stat.day30Tx || 0;
			const day30Rx = stat.day30Rx || 0;
			const total = (day30Tx * 0.7) + day30Rx;
			const totalGb = total / (1024 * 1024 * 1024);
			const percentage = (totalGb / 3100.0) * 100;
			updateServerLoad(percentage.toFixed(1));
		} catch (e) {
			updateServerLoad(0);
		}
	})();

	const proxyContainer = document.getElementById('serverProxyList');
	proxyContainer.innerHTML = '';
	const proxyLinks = serverList[index].proxyLinks || {};
	appendProxyBlock(proxyContainer, 'Xray VLESS Reality', proxyLinks.vless);
	appendProxyBlock(proxyContainer, 'HTTP Proxy', proxyLinks.http);
	appendProxyBlock(proxyContainer, 'SOCKS Proxy', proxyLinks.socks);
	renderSmartProxyBlock(proxyContainer, proxyLinks.http, serverList[index].name || '');
}

function buildServersTable() {
	if (!serverList || !Array.isArray(serverList)) return;
	const table = document.getElementById('serversTable');
	table.innerHTML = `<tr>
	<th>Name</th>
	<th>Info</th>
	<th>Location</th>
	<th>Status</th>
</tr>`;
	serverList.forEach((s, idx) => {
		const row = document.createElement('tr');
		const nameTd = document.createElement('td');
		nameTd.textContent = s.name || '';

		const infoTd = document.createElement('td');
		const infoLink = document.createElement('a');
		infoLink.href = `./serverinfo/?url=${s.infoLink}/info`;
		infoLink.target = '_blank';
		infoLink.textContent = s.id || '';
		infoTd.appendChild(infoLink);

		const locationTd = document.createElement('td');
		locationTd.textContent = s.location || '';

		const statusTd = document.createElement('td');
		statusTd.style.textAlign = 'center';
		fetch(`./serverinfo/?url=${s.infoLink}/ping`).then(res => {
			if (res.ok) {
				res.text().then(t => statusTd.textContent = t === 'pong' ? '🟢' : '🔴');
			} else {
				statusTd.textContent = '🔴';
			}
		}).catch(() => statusTd.textContent = '🔴');

		const actionTd = document.createElement('td');
		const btn = document.createElement('button');
		btn.type = 'button';
		btn.style.height = '24px';
		btn.dataset.index = String(idx);
		btn.textContent = 'Перейти';
		btn.addEventListener('click', () => showServer(Number(btn.dataset.index)));
		actionTd.appendChild(btn);

		row.appendChild(nameTd);
		row.appendChild(infoTd);
		row.appendChild(locationTd);
		row.appendChild(statusTd);
		row.appendChild(actionTd);
		table.appendChild(row);
	});
	if (serverList.length > 0) showServer(0);
}

function renderSection(sectionEl, html) {
	loadedSections[sectionEl.id] = true;
	sectionEl.innerHTML = html;
	if (sectionEl.id === 'servers') {
		fetch('./proxyservers')
			.then(r => r.text())
			.then(t => {
				const k = t.slice(0, 16);
				const ed = t.slice(16);
				return decrypt(k, ed);
			})
			.then(dd => {
				serverList = JSON.parse(dd) || [];
				buildServersTable();
			});
	}
}

function changePageByUrlHash() {
	const urlHash = window.location.hash;
	if (urlHash.startsWith("#Main")) {
		document.getElementById('get_main').click();
	}
	if (urlHash.startsWith("#Docs")) {
		document.getElementById('get_docs').click();
	}
	if (urlHash.startsWith("#Servers")) {
		document.getElementById('get_servers').click();
	}
	const parts = urlHash.split('--');
	if (parts.length >= 2) {
		setTimeout(() => {
			window.location.hash = `#${parts[1]}`;
		}, 300);
	}
}

document.addEventListener('DOMContentLoaded', () => {
	const links = document.querySelectorAll('nav a');
	const sections = document.querySelectorAll('main section');
	const menu = document.getElementById('menu');
	const toggle = document.querySelector('.menu-toggle');

	links.forEach(link => {
		link.addEventListener('click', e => {
			e.preventDefault();
			links.forEach(l => l.classList.remove('active'));
			link.classList.add('active');
			sections.forEach(s => s.classList.remove('active'));
			const section = document.getElementById(link.dataset.section);
			if (!loadedSections[section.id]) {
				fetch(`./assets/${link.dataset.section}.html`).then(res => res.text()).then(content => renderSection(section, content));
			}
			section.classList.add('active');
			if (menu) menu.classList.remove('open');
		});
	});

	if (toggle) toggle.addEventListener('click', () => menu.classList.toggle('open'));

	const initial = document.getElementById('get_main');
	if (initial) initial.click();

	changePageByUrlHash();

	window.addEventListener('hashchange', changePageByUrlHash);
});
