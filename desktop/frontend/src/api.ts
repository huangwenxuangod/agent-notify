export type AgentStatus={id:string;name:string;installed:boolean;hook_installed:boolean;error?:string};
export type EventRecord={id:string;timestamp:string;agent:string;event:string;title:string;body:string;result:string};
export type Channel={Enabled:boolean;ClickToFocus?:boolean;WebhookURL?:string;TopicURL?:string;SigningSecret?:string;AccessToken?:string};
export type AgentNotify={Events:string[];Channels:Record<string,Channel>};
export type RemoteDelivery={Feishu:Channel;Wechat:Channel;WechatWork:Channel;DingTalk:Channel;Bark:Channel;Ntfy:Channel;Slack:Channel};
export type Config={Version:number;Agent:Record<string,{Enabled:boolean;InstallScope:string}>;Notify:Record<string,AgentNotify>;Remote:RemoteDelivery;Behavior:{DedupeSeconds:number;SendTimeoutSeconds:number;Locale:string}};
export type AutostartStatus={Supported:boolean;Enabled:boolean;Platform:string;Path?:string;Error?:string};
export type HookRuntimeStatus={installed:boolean;last_event_at?:string;last_event?:string};

declare global { interface Window { go?: { main?: { App?: {
  Scan:()=>Promise<AgentStatus[]>;
  AutoSetup:()=>Promise<AgentStatus[]>;
  Install:(agents:string[],scope:string)=>Promise<unknown>;
  Uninstall:(agents:string[],scope:string)=>Promise<unknown>;
  Events:()=>Promise<EventRecord[]>;
  Config:()=>Promise<Config>;
  SaveConfig:(config:Config)=>Promise<void>;
  SendTest:(agent:string)=>Promise<void>;
  SendTestChannel:(channel:string)=>Promise<void>;
	TestSystemNotification:()=>Promise<void>;
	CodexHookStatus:()=>Promise<HookRuntimeStatus>;
  OpenCodexHookReview:()=>Promise<void>;
  PauseOneHour:()=>Promise<void>;
  ResumeNotifications:()=>Promise<void>;
  Autostart:()=>Promise<AutostartStatus>;
  SetAutostart:(enabled:boolean)=>Promise<void>;
  ClickToFocus:()=>Promise<boolean>;
	SetClickToFocus:(enabled:boolean)=>Promise<void>;
	SystemNotifications:()=>Promise<boolean>;
	SetSystemNotifications:(enabled:boolean)=>Promise<void>;
} } } } }

const app=()=>window.go?.main?.App;
const call=<T>(name:keyof NonNullable<ReturnType<typeof app>>,...args:unknown[])=>{
  const fn=app()?.[name] as ((...values:unknown[])=>Promise<T>)|undefined;
  if(!fn)throw new Error('桌面运行时未连接');
  return fn(...args);
};
export const scan=()=>call<AgentStatus[]>('Scan');
export const autoSetup=()=>call<AgentStatus[]>('AutoSetup');
export const install=(agents:string[],scope:string)=>call<unknown>('Install',agents,scope);
export const uninstall=(agents:string[],scope:string)=>call<unknown>('Uninstall',agents,scope);
export const events=()=>call<EventRecord[]>('Events');
export const config=()=>call<Config>('Config');
export const saveConfig=(value:Config)=>call<void>('SaveConfig',value);
export const sendTest=(agent:string)=>call<void>('SendTest',agent);
export const sendTestChannel=(channel:string)=>call<void>('SendTestChannel',channel);
export const testSystemNotification=()=>call<void>('TestSystemNotification');
export const codexHookStatus=()=>call<HookRuntimeStatus>('CodexHookStatus');
export const openCodexHookReview=()=>call<void>('OpenCodexHookReview');
export const pauseOneHour=()=>call<void>('PauseOneHour');
export const resumeNotifications=()=>call<void>('ResumeNotifications');
export const autostart=()=>call<AutostartStatus>('Autostart');
export const setAutostart=(enabled:boolean)=>call<void>('SetAutostart',enabled);
export const clickToFocus=()=>call<boolean>('ClickToFocus');
export const setClickToFocus=(enabled:boolean)=>call<void>('SetClickToFocus',enabled);
export const systemNotifications=()=>call<boolean>('SystemNotifications');
export const setSystemNotifications=(enabled:boolean)=>call<void>('SetSystemNotifications',enabled);
