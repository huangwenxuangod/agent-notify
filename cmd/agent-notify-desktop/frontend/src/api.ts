export type AgentStatus={id:string;name:string;installed:boolean;hook_installed:boolean;error?:string};
export type EventRecord={id:string;timestamp:string;agent:string;event:string;title:string;body:string;result:string};
declare global { interface Window { go?: { main?: { App?: { Scan:()=>Promise<AgentStatus[]>; Install:(agents:string[],scope:string)=>Promise<unknown>; Events:()=>Promise<EventRecord[]> } } } } }
const app=()=>window.go?.main?.App;
export async function scan():Promise<AgentStatus[]>{const fn=app()?.Scan;if(!fn)throw new Error('Wails runtime unavailable');return fn();}
export async function install(agents:string[],scope:string){const fn=app()?.Install;if(!fn)throw new Error('Wails runtime unavailable');return fn(agents,scope);}
export async function events():Promise<EventRecord[]>{const fn=app()?.Events;if(!fn)throw new Error('Wails runtime unavailable');return fn();}
