const { useState, useEffect, useRef, useMemo, Fragment, useCallback } = React;

// ─── SVG Icons (inline, no external deps) ───
const Icons = {
  msg: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m3 21 1.9-5.7a8.5 8.5 0 1 1 3.8 3.8z"/></svg>,
  send: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/></svg>,
  settings: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>,
  search: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>,
  refresh: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>,
  check: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M20 6 9 17l-5-5"/></svg>,
  checks: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 7 17l-5-5"/><path d="m22 10-7.5 7.5L13 16"/></svg>,
  clock: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>,
  x: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>,
  loader: <svg className="spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>,
  rocket: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"/><path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/></svg>,
  save: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>,
  phone: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/></svg>,
  bot: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/></svg>,
  hash: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="4" x2="20" y1="9" y2="9"/><line x1="4" x2="20" y1="15" y2="15"/><line x1="10" x2="8" y1="3" y2="21"/><line x1="16" x2="14" y1="3" y2="21"/></svg>,
  key: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.3 9.3"/><path d="m18 5 3 3"/></svg>,
  globe: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg>,
  zap: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>,
  users: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>,
  okCircle: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><path d="m9 11 3 3L22 4"/></svg>,
  back: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m15 18-6-6 6-6"/></svg>,
  megaphone: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m3 11 18-5v12L3 13v-2z"/><path d="M11.6 16.8a3 3 0 1 1-5.8-1.6"/></svg>,
  target: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>,
  shield: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>,
  timer: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="10" x2="14" y1="2" y2="2"/><line x1="12" x2="15" y1="14" y2="11"/><circle cx="12" cy="14" r="8"/></svg>,
  inbox: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>,
  sendOut: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/></svg>,
  ban: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>,
  calendar: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/></svg>,
  activity: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>,
  sun: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>,
  moon: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>,
  camera: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3z"/><circle cx="12" cy="13" r="3"/></svg>,
  video: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>,
  mic: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>,
  file: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>,
  paperclip: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>,
};
const I = ({name, size=20, style={}}) => <span style={{display:'inline-flex',width:size,height:size,...style}}>{Icons[name]}</span>;

// ─── Utilities ───
const COLORS = ['#10b981','#3b82f6','#8b5cf6','#f43f5e','#f59e0b','#06b6d4','#6366f1','#d946ef'];
const getColor = s => { if(!s) return COLORS[0]; let h=0; for(let i=0;i<s.length;i++) h=s.charCodeAt(i)+((h<<5)-h); return COLORS[Math.abs(h)%COLORS.length]; };
const getInitials = (name,phone) => { if(name?.length) { const p=name.trim().split(' '); return p.length>1?(p[0][0]+p[1][0]).toUpperCase():name.slice(0,2).toUpperCase();} return phone?phone.slice(-2):'??'; };
const fmtTime = ts => { if(!ts)return''; const d=new Date(ts); return isNaN(d)?ts:d.toLocaleTimeString('en-IN',{hour:'2-digit',minute:'2-digit',hour12:true}); };
const fmtDate = ts => { if(!ts)return''; const d=new Date(ts); if(isNaN(d))return''; const n=new Date(); if(d.toDateString()===n.toDateString())return'Today'; const y=new Date(n);y.setDate(n.getDate()-1); if(d.toDateString()===y.toDateString())return'Yesterday'; return d.toLocaleDateString('en-IN',{day:'2-digit',month:'short',year:'numeric'}); };
const MEDIA_LABELS = {image:'Photo',video:'Video',audio:'Audio',document:'Document',sticker:'Sticker',location:'Location',reaction:'Reaction',button:'Button',interactive:'Interactive',order:'Order'};
const mediaLabel = (type) => MEDIA_LABELS[type] || type;
const isMediaText = (msg) => /^\[(IMAGE|VIDEO|AUDIO|DOCUMENT|STICKER|LOCATION|REACTION|ORDER)\]/.test(msg);
const cleanMsg = (msg, type) => { if(!msg) return ''; if(isMediaText(msg)) return ''; return msg; };
const stripEmoji = (str) => { if(!str) return ''; return str.replace(/[\u{1F000}-\u{1FFFF}\u{2600}-\u{27BF}\u{FE00}-\u{FEFF}\u{1F900}-\u{1F9FF}]/gu,'').replace(/[^\x20-\x7E\u0900-\u097F\u0080-\u00FF]/g,'').trim(); };

// ─── Main App ───
function App() {
  const [tab, setTab] = useState('chats');
  const [theme,setTheme] = useState(()=>localStorage.getItem('wa-theme')||'dark');
  const [live, setLive] = useState(false);
  const [msgs, setMsgs] = useState([]);
  const [statuses, setStatuses] = useState([]);
  const [selPhone, setSelPhone] = useState(null);
  const [search, setSearch] = useState('');
  const [config, setConfig] = useState({});
  // Read state stored in backend DB — shared across all browsers
  const lastSeenRef = useRef(JSON.parse(localStorage.getItem('wa-lastseen')||'{}'));
  const pendingSyncRef = useRef({}); // batch pending writes
  const syncTimerRef = useRef(null);

  // Track current selPhone always fresh (avoids stale closure)
  const selPhoneRef = useRef(null);

  // Sync pending read states to backend (debounced)
  const flushReadState = useCallback(() => {
    const batch = pendingSyncRef.current;
    if(Object.keys(batch).length === 0) return;
    pendingSyncRef.current = {};
    fetch('/api/readstate', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(batch)}).catch(()=>{});
  }, []);

  const markSeen = useCallback((phone) => {
    if(!phone) return;
    const ts = new Date().toISOString();
    lastSeenRef.current[phone] = ts;
    localStorage.setItem('wa-lastseen', JSON.stringify(lastSeenRef.current));
    // Queue backend sync
    pendingSyncRef.current[phone] = ts;
    clearTimeout(syncTimerRef.current);
    syncTimerRef.current = setTimeout(flushReadState, 1500);
  }, [flushReadState]);

  // Mark contact as seen when selected — also marks PREVIOUS chat as seen
  const handleSelectPhone = useCallback((phone) => {
    if(selPhoneRef.current) markSeen(selPhoneRef.current);
    if(phone) markSeen(phone);
    selPhoneRef.current = phone;
    setSelPhone(phone);
  }, [markSeen]);

  // When switching tabs, mark current chat as seen
  const handleSetTab = useCallback((t) => {
    if(selPhoneRef.current) markSeen(selPhoneRef.current);
    setTab(t);
  }, [markSeen]);

  const fetchData = useCallback(async () => {
    try {
      const [mr, sr] = await Promise.all([fetch('/api/messages'), fetch('/api/statuses')]);
      if (mr.ok && sr.ok) { setMsgs(await mr.json()||[]); setStatuses(await sr.json()||[]); setLive(true); }
      else setLive(false);
    } catch { setLive(false); }
  }, []);

  useEffect(() => {
    // Load read state from backend DB first (works across browsers)
    fetch('/api/readstate').then(r=>r.json()).then(serverState=>{
      // Merge: take the LATER timestamp between local and server
      const merged = {...lastSeenRef.current};
      for(const [phone, ts] of Object.entries(serverState||{})) {
        if(!merged[phone] || ts > merged[phone]) merged[phone] = ts;
      }
      lastSeenRef.current = merged;
      localStorage.setItem('wa-lastseen', JSON.stringify(merged));
    }).catch(()=>{});

    fetchData();
    fetch('/api/config').then(r=>r.json()).then(d=>setConfig(d||{})).catch(()=>{});
    const iv = setInterval(fetchData, 5000);
    // SSE real-time
    let es;
    try {
      es = new EventSource('/api/events');
      es.addEventListener('new_message', () => fetchData());
      es.addEventListener('status_update', () => fetchData());
      es.addEventListener('broadcast_complete', () => fetchData());
      es.onerror = () => {};
    } catch {}
    return () => { clearInterval(iv); es?.close(); clearTimeout(syncTimerRef.current); flushReadState(); };
  }, [fetchData, flushReadState]);

  useEffect(()=>{
    document.documentElement.classList.toggle('light',theme==='light');
    localStorage.setItem('wa-theme',theme);
  },[theme]);
  const toggleTheme=()=>setTheme(t=>t==='dark'?'light':'dark');

  // Auto-mark currently open chat as seen whenever new data arrives
  useEffect(()=>{
    if(selPhoneRef.current) {
      markSeen(selPhoneRef.current);
    }
  },[msgs, markSeen]);

  const contacts = useMemo(() => {
    const map = {};
    const sorted = [...msgs].sort((a,b) => new Date(a.timestamp)-new Date(b.timestamp));
    sorted.forEach(m => {
      if(!map[m.phone]) map[m.phone] = {phone:m.phone,push_name:'',messages:[],lastMsg:m,lastTime:m.timestamp};
      map[m.phone].messages.push(m);
      map[m.phone].lastMsg = m;
      map[m.phone].lastTime = m.timestamp;
      if(m.direction==='incoming' && m.push_name) map[m.phone].push_name = m.push_name;
    });
    const list = Object.values(map).sort((a,b)=>new Date(b.lastTime)-new Date(a.lastTime))
      .filter(c => { const q=search.toLowerCase(); return c.phone.includes(q)||c.push_name.toLowerCase().includes(q); });
    // Compute unread counts
    list.forEach(c => {
      const seen = lastSeenRef.current[c.phone];
      c.unread = seen ? c.messages.filter(m => m.direction==='incoming' && new Date(m.timestamp)>new Date(seen)).length
        : c.messages.filter(m => m.direction==='incoming').length;
    });
    return list;
  }, [msgs, search]);

  const statusMap = useMemo(() => { const m={}; statuses.forEach(s=>{if(!m[s.message_id])m[s.message_id]=[];m[s.message_id].push(s.status)}); return m; }, [statuses]);
  const getBest = id => { const l=statusMap[id]||[]; if(l.includes('read'))return'read'; if(l.includes('delivered'))return'delivered'; if(l.includes('sent'))return'sent'; if(l.includes('failed'))return'failed'; return''; };

  return (
    <div className="app">
      <div className="sidebar">
        <div className="nav-items">
          {[['chats','msg','Messages'],['broadcast','rocket','Broadcast'],['autoreply','bot','Auto-Reply'],['followups','clock','Follow-Ups'],['config','settings','Settings']].map(([id,icon,label])=>
            <button key={id} className={`nav-btn ${tab===id?'active':''}`} onClick={()=>handleSetTab(id)} title={label}><I name={icon} size={20}/></button>
          )}
        </div>
      </div>
      <div className={`content ${tab==='chats'&&selPhone?'chat-open':''}`}>
        <div className="topbar">
          <div className={`status-dot ${live?'live':'dead'}`} title={live?'Connected':'Disconnected'}/>
          <button className="theme-toggle" onClick={toggleTheme} title={theme==='dark'?'Switch to Light':'Switch to Dark'}>
            <I name={theme==='dark'?'sun':'moon'} size={18}/>
          </button>
        </div>
        <div className="content-body">
          {tab==='chats' && <ChatsView contacts={contacts} sel={selPhone} setSel={handleSelectPhone} onBack={()=>{markSeen(selPhoneRef.current);selPhoneRef.current=null;setSelPhone(null);}} search={search} setSearch={setSearch} msgCount={msgs.length} getBest={getBest} onRefresh={fetchData}/>}
          {tab==='broadcast' && <BroadcastView config={config}/>}
          {tab==='autoreply' && <AutoReplyView/>}
          {tab==='followups' && <FollowUpView/>}
          {tab==='config' && <ConfigView config={config} setConfig={setConfig}/>}
        </div>
      </div>
    </div>
  );
}

// ─── Chats View ───
function ChatsView({contacts,sel,setSel,onBack,search,setSearch,msgCount,getBest,onRefresh}) {
  return (
    <div style={{display:'flex',flex:1,height:'100%',overflow:'hidden'}}>
      <div className="contact-list">
        <div className="contact-header">
          <h2>Messages</h2>
          <div className="search-box">
            <I name="search" size={14}/>
            <input placeholder="Search contacts..." value={search} onChange={e=>setSearch(e.target.value)}/>
          </div>
        </div>
        <div className="contacts-scroll">
          {contacts.length===0 ? <div className="loader"><I name="loader" size={18}/><span>Syncing data...</span></div> :
            contacts.map(c => {
              const active = c.phone===sel;
              const tb = c.lastMsg.message_type!=='text'?c.lastMsg.message_type:null;
              const hasUnread = c.unread > 0 && !active;
              return (
                <div key={c.phone} className={`contact-item ${active?'active':''} ${hasUnread?'unread':''}`} onClick={()=>setSel(c.phone)}>
                  <div className="avatar" style={{backgroundColor:getColor(c.phone)}}>{getInitials(c.push_name,c.phone)}</div>
                  <div className="contact-info">
                    <div className="contact-top">
                      <span className={`contact-name ${hasUnread?'unread':''}`}>{c.push_name||c.phone}</span>
                      <span className={`contact-time ${hasUnread?'unread':''}`}>{fmtTime(c.lastTime)}</span>
                    </div>
                    <div style={{display:'flex',alignItems:'center',gap:6}}>
                      <p className="contact-preview" style={{flex:1}}>
                        {tb ? <span style={{color:'var(--text3)',display:'flex',alignItems:'center',gap:4}}>
                            <I name={tb==='image'?'camera':tb==='video'?'video':tb==='audio'?'mic':tb==='document'?'file':'paperclip'} size={12} style={{opacity:0.7}}/>
                            {mediaLabel(tb)}
                          </span>
                          : <span>{stripEmoji(c.lastMsg.message)?.substring(0,40)||''}</span>}
                      </p>
                      {hasUnread && <span className="unread-badge">{c.unread}</span>}
                    </div>
                  </div>
                </div>
              );
            })
          }
        </div>
      </div>
      <div className="chat-area">
        <div className="chat-bg"/>
        {sel ? <ChatThread contact={contacts.find(c=>c.phone===sel)} getBest={getBest} onRefresh={onRefresh} onBack={onBack}/> :
          <div className="empty-state">
            <div className="empty-icon"><I name="zap" size={28}/></div>
            <h2>Workspace Hub</h2>
            <p>Select a chat to start messaging</p>
            <div className="stats-grid">
              <div className="stat-card"><div className="num">{msgCount}</div><div className="label">Total Messages</div></div>
              <div className="stat-card"><div className="num">{contacts.length}</div><div className="label">Active Contacts</div></div>
            </div>
          </div>
        }
      </div>
    </div>
  );
}

// ─── Chat Thread ───
function ChatThread({contact,getBest,onRefresh,onBack}) {
  const [input,setInput] = useState('');
  const [sending,setSending] = useState(false);
  const endRef = useRef(null);
  useEffect(()=>{endRef.current?.scrollIntoView({behavior:'smooth'})},[contact?.messages]);

  const doSend = async () => {
    if(!input.trim()||!contact) return;
    setSending(true);
    try { await fetch('/api/reply',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({to:contact.phone,message:input})}); setInput(''); onRefresh(); }
    catch(e){console.error(e)} finally{setSending(false)}
  };

  if(!contact) return null;
  let lastDate='';
  return (
    <div style={{display:'flex',flexDirection:'column',height:'100%',position:'relative',zIndex:1}}>
      <div className="chat-header">
        <div className="chat-header-info">
          <button className="back-btn" onClick={onBack}><I name="back" size={18}/></button>
          <div className="avatar" style={{backgroundColor:getColor(contact.phone),width:42,height:42,fontSize:13}}>{getInitials(contact.push_name,contact.phone)}</div>
          <div><h2>{contact.push_name||contact.phone}</h2><p style={{display:'flex',alignItems:'center',gap:4}}><I name="phone" size={10}/>+{contact.phone}</p></div>
        </div>
        <button className="refresh-btn" onClick={onRefresh}><I name="refresh" size={14}/></button>
      </div>
      <div className="messages-scroll">
        {contact.messages.map((m,i)=>{
          const ds=fmtDate(m.meta_timestamp||m.timestamp); const showDate=ds!==lastDate; if(showDate)lastDate=ds;
          const out=m.direction==='outgoing'; const st=out?getBest(m.message_id||m.id?.toString()):null;
          return (
            <Fragment key={m.id||i}>
              {showDate && <div className="date-sep"><span>{ds}</span></div>}
              <div className={`msg-row ${out?'out':'in'}`}>
                <div className="msg-bubble">
                  {m.message_type!=='text' && <div style={{fontSize:13,opacity:.85,marginBottom:cleanMsg(m.message,m.message_type)?4:0}}>{mediaLabel(m.message_type)}</div>}
                  {cleanMsg(m.message,m.message_type) ? <div>{cleanMsg(m.message,m.message_type)}</div> : (m.message_type==='location' ? <div>📍 {m.location_lat}, {m.location_lng}</div> : m.message_type==='text' ? <div>{m.message}</div> : null)}
                  <div className="msg-meta">
                    <span className="time">{fmtTime(m.meta_timestamp||m.timestamp)}</span>
                    {out && <span>{st==='read'?<I name="checks" size={14}/>:st==='delivered'?<I name="checks" size={14} style={{opacity:.6}}/>:st==='sent'?<I name="check" size={14}/>:st==='failed'?<I name="x" size={14}/>:<I name="clock" size={10}/>}</span>}
                  </div>
                </div>
              </div>
            </Fragment>
          );
        })}
        <div ref={endRef} style={{height:16}}/>
      </div>
      <div className="chat-input-area">
        <div className="chat-input-wrap">
          <textarea value={input} onChange={e=>setInput(e.target.value)} onKeyDown={e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();doSend()}}} placeholder="Type a message..." rows={1}/>
          <button className="send-btn" onClick={doSend} disabled={!input.trim()||sending}>
            {sending?<I name="loader" size={16}/>:<I name="send" size={16}/>}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Broadcast View ───
function BroadcastView({config}) {
  const [tpls,setTpls]=useState([]); const [selTpl,setSelTpl]=useState(null);
  const [nums,setNums]=useState(''); const [vars,setVars]=useState({});
  const [state,setState]=useState({is_running:false,total:0,processed:0,success:0,failed:0});
  const [loading,setLoading]=useState(false);

  useEffect(()=>{fetch('/api/templates').then(r=>r.json()).then(d=>setTpls(d||[])).catch(()=>{});},[]);
  useEffect(()=>{if(tpls.length>0&&config?.templateName&&!selTpl){const t=tpls.find(x=>x.name===config.templateName);if(t){setSelTpl(t);setVars({1:config.var1||'',2:config.var2||'',3:config.var3||'',4:config.var4||''});}}},[tpls,config,selTpl]);

  useEffect(()=>{
    let iv; const poll=async()=>{try{const r=await fetch('/api/broadcast/status?t='+Date.now());if(r.ok){const d=await r.json();setState(d);}}catch{}};
    poll(); iv=setInterval(poll,1500); return()=>clearInterval(iv);
  },[]);

  const vc=useMemo(()=>{if(!selTpl)return 0;const b=selTpl.components?.find(c=>c.type==='BODY'||c.type==='body');return b?(b.text.match(/{{/g)||[]).length:0;},[selTpl]);

  const doSend=async()=>{
    const nl=nums.split('\n').map(n=>n.trim()).filter(Boolean);
    if(!nl.length||!selTpl)return;
    const params=Array.from({length:vc},(_,i)=>vars[i+1]||'');
    setLoading(true);
    try{await fetch('/api/broadcast',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({numbers:nl,template:selTpl.name,language:selTpl.language,params})});setNums('');}
    catch(e){console.error(e)}finally{setLoading(false)}
  };

  const pct = state.total>0?Math.round(((state.success+state.failed)/state.total)*100):0;

  return (
    <div className="page-scroll"><div className="page-inner">
      <div className="page-title"><I name="rocket" size={24}/>Mass Broadcast</div>
      <p className="page-subtitle">Send verified Meta templates to your contacts at scale.</p>
      <div className="grid-2">
        <div><div className="card">
          <div className="card-title">Message Config</div>
          <div className="form-group">
            <label className="form-label">Template</label>
            <select className="form-select" value={selTpl?selTpl.name:''} onChange={e=>setSelTpl(tpls.find(t=>t.name===e.target.value))}>
              <option value="">-- Select template --</option>
              {tpls.map(t=><option key={t.name} value={t.name}>{t.name} ({t.language})</option>)}
            </select>
          </div>
          {vc>0 && <div style={{background:'var(--surface2)',border:'1px solid var(--border)',borderRadius:10,padding:16}}>
            <div className="form-label" style={{marginBottom:12}}>Template Variables</div>
            {Array.from({length:vc}).map((_,i)=><div className="var-row" key={i}>
              <span className="var-tag">{`{{${i+1}}}`}</span>
              <input className="form-input" style={{flex:1}} value={vars[i+1]||''} onChange={e=>setVars({...vars,[i+1]:e.target.value})} placeholder="Enter value..."/>
            </div>)}
          </div>}
        </div></div>
        <div><div className="card">
          <div className="card-title">Audience & Send</div>
          <div className="form-group">
            <div style={{display:'flex',justifyContent:'space-between',alignItems:'baseline',marginBottom:8}}>
              <label className="form-label" style={{margin:0}}>Recipients</label>
              <span style={{fontSize:10,color:'var(--text4)'}}>Include country code</span>
            </div>
            <textarea className="form-textarea" value={nums} onChange={e=>setNums(e.target.value)} placeholder={"919876543210\n918888888888"}/>
          </div>
          <button className="primary-btn" onClick={doSend} disabled={loading||state.is_running||!selTpl||!nums.trim()}>
            {loading?<I name="loader" size={16}/>:<I name="send" size={16}/>}
            {state.is_running?'Campaign running...':'Start Campaign'}
          </button>
        </div>
        {(state.total>0||state.is_running) && <div className="progress-card">
          <div className="progress-header">
            <span className="label">{state.is_running?<><I name="loader" size={16}/>In Progress</>:<><I name="okCircle" size={16}/>Complete</>}</span>
            <span className="pct">{pct}%</span>
          </div>
          <div className="progress-bar">
            <div className="ok" style={{width:`${(state.success/Math.max(state.total,1))*100}%`}}/>
            <div className="fail" style={{width:`${(state.failed/Math.max(state.total,1))*100}%`}}/>
          </div>
          <div className="progress-stats">
            <div className="p-stat"><div className="num" style={{color:'var(--text)'}}>{state.total}</div><div className="lbl" style={{color:'var(--text4)'}}>Total</div></div>
            <div className="p-stat"><div className="num" style={{color:'var(--emerald)'}}>{state.success}</div><div className="lbl" style={{color:'var(--emerald)'}}>Delivered</div></div>
            <div className="p-stat"><div className="num" style={{color:'var(--red)'}}>{state.failed}</div><div className="lbl" style={{color:'var(--red)'}}>Failed</div></div>
          </div>
        </div>}
        </div>
      </div>
    </div></div>
  );
}

// ─── Config View ───
function ConfigView({config,setConfig}) {
  const [settings,setSettings]=useState({auto_reply_enabled:'false',auto_reply_message:''});
  const [saving,setSaving]=useState(false);
  const [status,setStatus]=useState('');
  const [ctab,setCtab]=useState('general');

  useEffect(()=>{fetch('/api/settings').then(r=>r.json()).then(d=>{if(d)setSettings(p=>({...p,...d}))}).catch(()=>{});},[]);

  const save=async()=>{
    setSaving(true);setStatus('Saving...');
    try{
      await Promise.all([
        fetch('/api/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(config)}),
        fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(settings)})
      ]);
      setStatus('✅ Saved successfully');setTimeout(()=>setStatus(''),3000);
    }catch{setStatus('❌ Error saving')}finally{setSaving(false)}
  };

  const Inp=({label,value,onChange,type='text'})=>(
    <div className="form-group"><label className="form-label">{label}</label>
      <input className="form-input" type={type} value={value||''} onChange={e=>onChange(e.target.value)}/></div>
  );
  const Tog=({label,desc,checked,onChange})=>(
    <div className="toggle-wrap">
      <button className={`toggle ${checked==='true'?'on':'off'}`} onClick={()=>onChange(checked==='true'?'false':'true')}/>
      <div><div className="toggle-label">{label}</div>{desc&&<div className="toggle-desc">{desc}</div>}</div>
    </div>
  );

  const tabs=[['general','General','hash'],['meta','Meta API','key'],['webhook','Webhook','globe']];

  return (
    <div className="page-scroll"><div className="page-inner" style={{maxWidth:800}}>
      <div className="config-header">
        <div><div className="page-title"><I name="settings" size={24}/>Settings</div></div>
        <button className="save-btn" onClick={save} disabled={saving}>
          {saving?<I name="loader" size={16}/>:<I name="save" size={16}/>}Save
        </button>
      </div>
      <div className="tabs">
        {tabs.map(([id,label,icon])=><button key={id} className={`tab-btn ${ctab===id?'active':''}`} onClick={()=>setCtab(id)}><I name={icon} size={16}/>{label}</button>)}
      </div>
      {status && <div className="status-msg">{status}</div>}
      <div style={{maxWidth:640}}>
        {ctab==='general' && <div className="card">
          <div className="card-title">Workspace</div>
          <Inp label="App Name" value={config.appName} onChange={v=>setConfig({...config,appName:v})}/>
          <Inp label="App ID" value={config.appId} onChange={v=>setConfig({...config,appId:v})}/>
          <Inp label="Broadcast Delay (ms)" value={config.broadcastDelay} onChange={v=>setConfig({...config,broadcastDelay:v})} type="number"/>
          <div style={{fontSize:12,color:'var(--text3)',marginTop:-10,marginBottom:8}}>Delay between each broadcast message in milliseconds (default: 200ms)</div>
          <Tog label="Debug Mode" desc="Enable extra logging in terminal" checked={config.debug} onChange={v=>setConfig({...config,debug:v})}/>
        </div>}
        {ctab==='meta' && <div className="card">
          <div className="card-title">Meta Credentials</div>
          <Inp label="API Version" value={config.apiVersion} onChange={v=>setConfig({...config,apiVersion:v})}/>
          <Inp label="System User Token" type="password" value={config.whatsappToken} onChange={v=>setConfig({...config,whatsappToken:v})}/>
          <div style={{display:'grid',gridTemplateColumns:'1fr 1fr',gap:16}}>
            <Inp label="Phone ID" value={config.phoneNumberId} onChange={v=>setConfig({...config,phoneNumberId:v})}/>
            <Inp label="WABA ID" value={config.wabaId} onChange={v=>setConfig({...config,wabaId:v})}/>
          </div>
        </div>}
        {ctab==='webhook' && <div className="card">
          <div className="card-title">Webhook</div>
          <Inp label="Webhook URL" value={config.webhookUrl} onChange={v=>setConfig({...config,webhookUrl:v})}/>
          <Inp label="Verify Token" value={config.verifyToken} onChange={v=>setConfig({...config,verifyToken:v})}/>
        </div>}
      </div>
    </div></div>
  );
}

// ─── Auto-Reply View ───
function AutoReplyView() {
  const [settings,setSettings]=useState({});
  const [saving,setSaving]=useState(false);
  const [status,setStatus]=useState('');
  useEffect(()=>{fetch('/api/settings').then(r=>r.json()).then(d=>{if(d)setSettings(d)}).catch(()=>{});},[]);
  const save=async()=>{setSaving(true);setStatus('Saving...');try{await fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(settings)});setStatus('✅ Saved');setTimeout(()=>setStatus(''),3000);}catch{setStatus('❌ Error')}finally{setSaving(false)}};
  const Tog=({label,desc,checked,onChange})=>(<div className="toggle-wrap"><button className={`toggle ${checked==='true'?'on':'off'}`} onClick={()=>onChange(checked==='true'?'false':'true')}/><div><div className="toggle-label">{label}</div>{desc&&<div className="toggle-desc">{desc}</div>}</div></div>);
  const ds=Number(settings.auto_reply_delay_seconds||300);
  const dl=ds>=60?Math.round(ds/60)+' min':ds+' sec';
  return (
    <div className="page-scroll"><div className="page-inner" style={{maxWidth:800}}>
      <div className="config-header">
        <div>
          <div className="page-title"><I name="bot" size={24}/>Auto-Reply</div>
          <p className="page-subtitle" style={{marginBottom:0}}>Automatically respond to incoming messages with a configurable delay</p>
        </div>
        <button className="save-btn" onClick={save} disabled={saving}>{saving?<I name="loader" size={16}/>:<I name="save" size={16}/>}Save Changes</button>
      </div>
      {status && <div className="status-msg">{status}</div>}

      <div style={{display:'grid',gridTemplateColumns:'1fr 1fr',gap:20,marginBottom:28}}>
        <div className="fu-stat">
          <div className="accent" style={{background:'linear-gradient(90deg,transparent,var(--emerald),transparent)'}}/>
          <div className="emoji"><I name="zap" size={24} style={{color:'var(--emerald)'}}/></div>
          <div className="title">Current Delay</div>
          <div style={{fontSize:22,fontWeight:700,color:'var(--emerald)',fontFamily:"'SF Mono',monospace",marginTop:4}}>{dl}</div>
        </div>
        <div className="fu-stat">
          <div className="accent" style={{background:'linear-gradient(90deg,transparent,var(--blue),transparent)'}}/>
          <div className="emoji"><I name="shield" size={24} style={{color:'var(--blue)'}}/></div>
          <div className="title">Cooldown</div>
          <div style={{fontSize:22,fontWeight:700,color:'var(--blue)',fontFamily:"'SF Mono',monospace",marginTop:4}}>{settings.auto_reply_interval_hours||12}h</div>
        </div>
      </div>

      <div className="card ar-card" style={{background:'var(--surface)',borderColor:'rgba(16,185,129,0.15)'}}>
        <div className="fu-card-head">
          <div className="card-title" style={{color:'var(--emerald)',margin:0,padding:0,border:'none',display:'flex',alignItems:'center',gap:8}}><I name="zap" size={14} style={{color:'var(--emerald)'}}/>Auto-Reply Configuration</div>
          <Tog label="" checked={settings.auto_reply_enabled} onChange={v=>setSettings({...settings,auto_reply_enabled:v})}/>
        </div>
        <div className="ar-delay-grid">
          <div className="form-group" style={{margin:0}}>
            <label className="form-label">Reply Delay (Seconds)</label>
            <input className="form-input" type="number" min="0" max="1800" value={settings.auto_reply_delay_seconds||'300'} onChange={e=>setSettings({...settings,auto_reply_delay_seconds:e.target.value})}/>
            <span className="ar-hint">300 = 5min · 1 = instant test</span>
          </div>
          <div className="form-group" style={{margin:0}}>
            <label className="form-label">Cooldown (Hours)</label>
            <input className="form-input" type="number" min="1" max="24" value={settings.auto_reply_interval_hours||'12'} onChange={e=>setSettings({...settings,auto_reply_interval_hours:e.target.value})}/>
            <span className="ar-hint">Block repeat reply to same user</span>
          </div>
        </div>
        <div className="form-group" style={{margin:'16px 0 0'}}>
          <label className="form-label">Auto-Reply Message</label>
          <textarea className="form-textarea" style={{minHeight:100,opacity:settings.auto_reply_enabled==='true'?1:.3,fontFamily:'inherit'}} value={settings.auto_reply_message||''} onChange={e=>setSettings({...settings,auto_reply_message:e.target.value})} disabled={settings.auto_reply_enabled!=='true'} placeholder="Write your auto-reply message here..."/>
        </div>
      </div>

      <div style={{background:'rgba(16,185,129,0.04)',border:'1px solid rgba(16,185,129,0.12)',borderRadius:12,padding:20,marginTop:20}}>
        <div style={{fontSize:11,fontWeight:600,color:'var(--emerald)',textTransform:'uppercase',letterSpacing:1.2,marginBottom:12}}>How It Works</div>
        <div style={{display:'flex',alignItems:'center',gap:8,flexWrap:'wrap'}}>
          {[
            ['inbox','User sends message','var(--surface)','var(--text)'],
            [null,'','transparent','var(--text4)'],
            ['timer',`Wait ${dl}`,'rgba(245,158,11,0.1)','var(--amber)'],
            [null,'','transparent','var(--text4)'],
            ['sendOut','Auto-reply sent','rgba(16,185,129,0.1)','var(--emerald)'],
            [null,'','transparent','var(--text4)'],
            ['ban',`Block ${settings.auto_reply_interval_hours||12}h`,'rgba(239,68,68,0.1)','var(--red)']
          ].map(([icon,text,bg,color],i)=>text?(
            <div key={i} style={{background:bg,border:`1px solid ${color}22`,borderRadius:10,padding:'8px 12px',display:'flex',alignItems:'center',gap:6,fontSize:12,color}}>
              <I name={icon} size={14} style={{color,flexShrink:0}}/>{text}
            </div>
          ):<span key={i} style={{color:'var(--text4)',fontSize:12,opacity:.5}}>→</span>)}
        </div>
      </div>
    </div></div>
  );
}

// ─── Follow-Up View ───
function FollowUpView() {
  const [settings,setSettings]=useState({});
  const [saving,setSaving]=useState(false);
  const [status,setStatus]=useState('');
  useEffect(()=>{fetch('/api/settings').then(r=>r.json()).then(d=>{if(d)setSettings(d)}).catch(()=>{});},[]);
  const save=async()=>{setSaving(true);setStatus('Saving...');try{await fetch('/api/settings',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(settings)});setStatus('✅ Saved');setTimeout(()=>setStatus(''),3000);}catch{setStatus('❌ Error')}finally{setSaving(false)}};
  const Tog=({label,checked,onChange})=>(<div className="toggle-wrap" style={{margin:0,padding:0}}><button className={`toggle ${checked==='true'?'on':'off'}`} onClick={()=>onChange(checked==='true'?'false':'true')}/>{label&&<div><div className="toggle-label">{label}</div></div>}</div>);
  const m1=Number(settings.followup_delay_minutes||480),m2=Number(settings.followup2_delay_minutes||720);
  const fmtM=m=>m>=60?Math.floor(m/60)+'h'+(m%60?' '+m%60+'m':''):m+' min';
  const ds=Number(settings.auto_reply_delay_seconds||300);
  const arLabel=ds>=60?Math.round(ds/60)+'m':ds+'s';

  const Step=({num,color,title,time,desc,statusText})=>(
    <div className="fu-step">
      <div className="fu-step-dot">
        <div className="fu-step-num" style={{background:`linear-gradient(135deg,${color},${color}88)`,boxShadow:`0 0 16px ${color}33`,border:`2px solid ${color}44`}}>{num}</div>
        <div className="fu-step-line" style={{background:`linear-gradient(180deg,${color}44,transparent)`}}/>
      </div>
      <div className="fu-step-content">
        <div className="fu-step-title">
          <span>{title}</span>
          <span className="fu-step-badge" style={{color,background:`${color}15`}}>{time}</span>
        </div>
        <div className="fu-step-desc">{desc}</div>
        {statusText&&<div style={{fontSize:11,marginTop:4}}>{statusText}</div>}
      </div>
    </div>
  );

  return (
    <div className="page-scroll"><div className="page-inner" style={{maxWidth:920}}>
      <div className="config-header">
        <div>
          <div className="page-title"><I name="clock" size={24}/>Scheduled Follow-Ups</div>
          <p className="page-subtitle" style={{marginBottom:0}}>Maximize conversions with automated multi-touch follow-ups within Meta's 24-hour window</p>
        </div>
        <button className="save-btn" onClick={save} disabled={saving}>{saving?<I name="loader" size={16}/>:<I name="save" size={16}/>}Save Changes</button>
      </div>
      {status && <div className="status-msg">{status}</div>}

      <div className="fu-stats">
        {[
          ['megaphone','Broadcast','Template sent to contacts','#3b82f6'],
          ['zap','Auto-Reply #1',`Triggers after ${arLabel}`,'#10b981'],
          ['target','Total Reach','3x promotions per user','#f59e0b']
        ].map(([icon,title,sub,color],i)=>(
          <div className="fu-stat" key={i}>
            <div className="accent" style={{background:`linear-gradient(90deg,transparent,${color},transparent)`}}/>
            <div className="emoji"><I name={icon} size={22} style={{color}}/></div>
            <div className="title">{title}</div>
            <div className="sub">{sub}</div>
          </div>
        ))}
      </div>

      <div className="fu-grid">
        <div className="fu-cards">
          <div className="card" style={{background:'var(--surface)',borderColor:'rgba(59,130,246,0.2)'}}>
            <div className="fu-card-head">
              <div className="card-title" style={{color:'#3b82f6',margin:0,padding:0,border:'none',display:'flex',alignItems:'center',gap:8}}><I name="calendar" size={14} style={{color:'#3b82f6'}}/>Follow-Up #2</div>
              <Tog checked={settings.followup_enabled} onChange={v=>setSettings({...settings,followup_enabled:v})}/>
            </div>
            <div className="fu-delay-row">
              <div className="form-group" style={{margin:0}}>
                <label className="form-label">Delay (Minutes)</label>
                <input className="form-input" type="number" min="1" max="1400" value={settings.followup_delay_minutes||'480'} onChange={e=>setSettings({...settings,followup_delay_minutes:e.target.value})}/>
              </div>
              <div className="fu-delay-val" style={{color:'#3b82f6'}}>{fmtM(m1)}</div>
            </div>
            <div className="form-group" style={{margin:0}}>
              <label className="form-label">Message</label>
              <textarea className="form-textarea" style={{minHeight:90,opacity:settings.followup_enabled==='true'?1:.3,fontFamily:'inherit'}} value={settings.followup_message||''} onChange={e=>setSettings({...settings,followup_message:e.target.value})} disabled={settings.followup_enabled!=='true'} placeholder="Write your follow-up message..."/>
            </div>
          </div>

          <div className="card" style={{background:'var(--surface)',borderColor:'rgba(139,92,246,0.2)'}}>
            <div className="fu-card-head">
              <div className="card-title" style={{color:'#8b5cf6',margin:0,padding:0,border:'none',display:'flex',alignItems:'center',gap:8}}><I name="calendar" size={14} style={{color:'#8b5cf6'}}/>Follow-Up #3</div>
              <Tog checked={settings.followup2_enabled} onChange={v=>setSettings({...settings,followup2_enabled:v})}/>
            </div>
            <div className="fu-delay-row">
              <div className="form-group" style={{margin:0}}>
                <label className="form-label">Delay (Minutes)</label>
                <input className="form-input" type="number" min="1" max="1400" value={settings.followup2_delay_minutes||'720'} onChange={e=>setSettings({...settings,followup2_delay_minutes:e.target.value})}/>
              </div>
              <div className="fu-delay-val" style={{color:'#8b5cf6'}}>{fmtM(m2)}</div>
            </div>
            <div className="form-group" style={{margin:0}}>
              <label className="form-label">Message</label>
              <textarea className="form-textarea" style={{minHeight:90,opacity:settings.followup2_enabled==='true'?1:.3,fontFamily:'inherit'}} value={settings.followup2_message||''} onChange={e=>setSettings({...settings,followup2_message:e.target.value})} disabled={settings.followup2_enabled!=='true'} placeholder="Write your last-chance message..."/>
            </div>
          </div>
        </div>

        <div>
          <div className="fu-timeline">
            <div className="fu-timeline-title">Campaign Timeline</div>
            <Step num="1" color="#3b82f6" title="Template Broadcast" time="T+0" desc="Bulk send to 20k+ contacts"/>
            <Step num="2" color="#10b981" title="Auto-Reply #1" time={`T+${arLabel}`} desc="Instant engagement on user reply"/>
            <Step num="3" color="#3b82f6" title="Follow-Up #2" time={`T+${fmtM(m1)}`} desc="Second promotional touch" statusText={settings.followup_enabled==='true'?'✅ Active':'⏸ Disabled'}/>
            <Step num="4" color="#8b5cf6" title="Follow-Up #3" time={`T+${fmtM(m2)}`} desc="Final conversion push" statusText={settings.followup2_enabled==='true'?'✅ Active':'⏸ Disabled'}/>
            <div className="fu-reach" style={{background:'linear-gradient(135deg,rgba(16,185,129,0.08),rgba(16,185,129,0.02))',border:'1px solid rgba(16,185,129,0.15)'}}>
              <div className="fu-reach-label" style={{color:'var(--emerald)'}}>Maximum Reach</div>
              <div className="fu-reach-value">3x <span>free promotions per user</span></div>
            </div>
          </div>
        </div>
      </div>
    </div></div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App/>);

