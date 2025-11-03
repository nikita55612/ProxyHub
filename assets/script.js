const sectionsLoad = {};
let servers;

function parseProxyUrl(url) {
	const urlObj = new URL(url);
	return {
		username: urlObj.username,
		password: urlObj.password,
		host: urlObj.hostname,
		port: urlObj.port
	};
}

function setServerLoad(value) {
	const slider = document.getElementById("serverLoadSlider");
	const counter = document.getElementById("serverLoadSliderCounter");
	const percent = Math.min(100, Math.max(0, value));

	slider.style.left = percent + '%';
	counter.textContent = percent + '%';
}

function showServer(index) {
	document.getElementById("serverContentName").innerText = servers[index].name;
	const serverContent = document.getElementById("serverContent");

	const providerName = document.getElementById("providerName");
	const tariffPlan = document.getElementById("tariffPlan");
	const speedLimit = document.getElementById("speedLimit");
	const trafficLimit = document.getElementById("trafficLimit");

	const a = document.createElement("a");
	a.href = servers[index].providerLink;
	a.innerText = servers[index].providerName;
	providerName.appendChild(a);

	tariffPlan.innerText = servers[index].plan;
	speedLimit.innerText = servers[index].speedRate;
	trafficLimit.innerText = servers[index].limit;

	document.getElementById("serverInfoTable").hidden = false;
	document.getElementById("serverLoadHeader").hidden = false;
	document.getElementById("serverLoadScaleContainer").hidden = false;
	fetch("")
		.then(res => res.text())
		.then(stat => {
			const statData = JSON.parse(stat);
			const day30Tx = statData["day30Tx"] || 0;
			const day30Rx = statData["day30Rx"] || 0;
			const total = day30Tx + day30Rx;
			const totalGb = total / (1024 * 1024 * 1024);
			const percentage = (totalGb / 3100.0) * 100;
			const roundedPercentage = percentage.toFixed(2);
			setServerLoad(roundedPercentage);
		})

	if (servers[index]["proxy"]["vless"] && servers[index]["proxy"]["vless"].length > 0) {
		const vlessHeader = document.createElement("h3");
		vlessHeader.innerText = "Xray VLESS Reality";

		serverContent.appendChild(vlessHeader);
		serverContent.appendChild(document.createElement("hr"))

		servers[index]["proxy"]["vless"].forEach((v) => {
			const p = document.createElement("p");
			p.classList.add("flex-jc-sb");
			const code = document.createElement("code");
			const pre = document.createElement("pre");
			pre.innerText = v;
			const copyBtn = document.createElement("button");
			copyBtn.innerText = "📋";
			copyBtn.className = "copyBtn";
			copyBtn.onclick = () => {
				navigator.clipboard.writeText(pre.innerText);
			};
			code.appendChild(pre);
			p.appendChild(code);

			p.appendChild(copyBtn);
			serverContent.appendChild(p);
		})
	}

	if (servers[index]["proxy"]["http"] && servers[index]["proxy"]["http"].length > 0) {
		const httpHeader = document.createElement("h3");
		httpHeader.innerText = "HTTP Proxy";

		serverContent.appendChild(httpHeader);
		serverContent.appendChild(document.createElement("hr"))

		servers[index]["proxy"]["http"].forEach((v) => {
			const p = document.createElement("p");
			p.classList.add("flex-jc-sb");
			const code = document.createElement("code");
			const pre = document.createElement("pre");
			pre.innerText = v;
			const copyBtn = document.createElement("button");
			copyBtn.innerText = "📋";
			copyBtn.className = "copyBtn";
			copyBtn.onclick = () => {
				navigator.clipboard.writeText(pre.innerText);
			};
			code.appendChild(pre);
			p.appendChild(code);

			p.appendChild(copyBtn);
			serverContent.appendChild(p);
		})
	}

	if (servers[index]["proxy"]["socks"] && servers[index]["proxy"]["socks"].length > 0) {
		const socksHeader = document.createElement("h3");
		socksHeader.innerText = "SOCKS Proxy";

		serverContent.appendChild(socksHeader);
		serverContent.appendChild(document.createElement("hr"))

		servers[index]["proxy"]["socks"].forEach((v) => {
			const p = document.createElement("p");
			p.classList.add("flex-jc-sb");
			const code = document.createElement("code");
			const pre = document.createElement("pre");
			pre.innerText = v;
			const copyBtn = document.createElement("button");
			copyBtn.innerText = "📋";
			copyBtn.className = "copyBtn";
			copyBtn.onclick = () => {
				navigator.clipboard.writeText(pre.innerText);
			};
			code.appendChild(pre);
			p.appendChild(code);

			p.appendChild(copyBtn);
			serverContent.appendChild(p);
		})
	}

	if (servers[index]["proxy"]["http"] && servers[index]["proxy"]["http"].length > 0) {
		const httpHeader = document.createElement("h3");
		httpHeader.innerText = "SmartProxy Servers";

		serverContent.appendChild(httpHeader);
		serverContent.appendChild(document.createElement("hr"))

		let smartProxyServers = "[SmartProxy Servers]\n";

		servers[index]["proxy"]["http"].forEach((v) => {
			try {
				const res = parseProxyUrl(v);
				smartProxyServers += `${res.host}:${res.port} [HTTP] [${servers[index].name}_${res.username}] [${res.username}] [${res.password}]\n`
			} catch (_) { }
		})

		const p = document.createElement("p");
		p.classList.add("flex-jc-sb");
		const code = document.createElement("code");
		const pre = document.createElement("pre");
		pre.innerText = smartProxyServers;
		const copyBtn = document.createElement("button");
		copyBtn.innerText = "📋";
		copyBtn.className = "copyBtn";
		copyBtn.onclick = () => {
			navigator.clipboard.writeText(pre.innerText);
		};
		code.appendChild(pre);
		p.appendChild(code);

		p.appendChild(copyBtn);
		serverContent.appendChild(p);
	}
}

function buildServersPage() {
	if (!servers) return;
	const serversTable = document.getElementById("serversTable");

	servers.forEach((s) => {
		const row = document.createElement("tr");
		const name = document.createElement("td");
		name.innerText = s.name;

		const info = document.createElement("td");
		const infoInner = document.createElement("a");
		infoInner.href = `${s.infoLink}/info`;
		infoInner.target = "_blank";
		infoInner.innerText = s.id;
		info.appendChild(infoInner);

		const location = document.createElement("td");
		location.innerText = s.location;

		const status = document.createElement("td");
		status.style = "text-align: center;";
		fetch(`${s.infoLink}/ping`)
			.then((res) => {
				console.log(`${s.infoLink}/ping`, res);
				if (res.status === 200) {
					status.innerText = "🟢";
					res.text()
						.then(t => {
							if (t !== "pong") {
								status.innerText = "🔴";
							}
						})
				} else {
					status.innerText = "🔴";
				}
			})
			.catch(e => {
				console.log(`${s.infoLink}/ping`, e);
				status.innerText = "🔴";
			})

		row.appendChild(name);
		row.appendChild(info);
		row.appendChild(location);
		row.appendChild(status);

		serversTable.appendChild(row);
	});

	if (servers.length > 0) {
		showServer(0)
	}
}

function drawSection(section, content) {
	sectionsLoad[section.id] = true;
	section.innerHTML = content;
	if (section.id === "servers") {
		fetch("./servers")
			.then(res => res.text())
			.then(data => {
				servers = JSON.parse(data);
				buildServersPage();
			});
	}
}

document.addEventListener('DOMContentLoaded', function () {
	const links = document.querySelectorAll("nav a");
	const sections = document.querySelectorAll("main section");
	const menu = document.getElementById("menu");
	const toggle = document.querySelector(".menu-toggle");

	links.forEach(link => {
		link.addEventListener("click", e => {
			e.preventDefault();
			links.forEach(l => l.classList.remove("active"));
			link.classList.add("active");
			sections.forEach(s => s.classList.remove("active"));
			const section = document.getElementById(link.dataset.section);
			if (!sectionsLoad[section.id]) {
				fetch(`./assets/${link.dataset.section}.html`)
					.then(res => res.text())
					.then(content => {
						drawSection(section, content);
					});
			}
			section.classList.add("active");
			menu.classList.remove("open");
		});
	});

	toggle.addEventListener("click", () => menu.classList.toggle("open"));
	document.getElementById("get_main").click();
});
