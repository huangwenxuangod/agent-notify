import {scan,install,events,AgentStatus,EventRecord} from './api';
const $=<T extends HTMLElement>(id:string)=>document.getElementById(id) as T;
const status=$<HTMLElement>('status'), agents=$<HTMLElement>('agents'), history=$<HTMLElement>('events');
function renderAgents(items:AgentStatus[]){agents.innerHTML=items.map(a=>`<label class="agent"><input type="checkbox" value="${a.id}" ${a.hook_installed?'checked':''}><span><strong>${a.name}</strong><small>${a.installed?'已检测到':'未检测到'}${a.hook_installed?' · Hook 已安装':''}</small></span></label>`).join('');status.textContent=`已发现 ${items.length} 个 Agent`}
function renderEvents(items:EventRecord[]){history.innerHTML=items.length?items.slice().reverse().map(e=>`<article><div><strong>${e.title||e.event}</strong><small>${e.agent} · ${new Date(e.timestamp).toLocaleString()}</small></div><p>${e.body||''}</p><span class="result">${e.result}</span></article>`).join(''):'<p class="muted">暂无事件</p>'}
async function refresh(){try{renderAgents(await scan());renderEvents(await events())}catch(e){status.textContent=String(e)}}
document.querySelector('#scan')?.addEventListener('click',refresh);
document.querySelector('#install')?.addEventListener('click',async()=>{const ids=Array.from(agents.querySelectorAll<HTMLInputElement>('input:checked')).map(x=>x.value);const scope=$<HTMLSelectElement>('scope').value;try{await install(ids,scope);await refresh()}catch(e){status.textContent=String(e)}});
refresh();
