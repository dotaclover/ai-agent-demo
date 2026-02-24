// State
let sessionId = localStorage.getItem('agent_session_id') || '';
let messages = []; // Array of {role, content, tool_calls, name}
let isBusy = false;

// Dom elements
const chatList = document.getElementById('chatList');
const chatView = document.getElementById('chatView');
const userInput = document.getElementById('userInput');
const sendBtn = document.getElementById('sendBtn');
const inputCard = document.querySelector('.input-card');
const welcomeScreen = document.getElementById('welcomeScreen');
const modal = document.getElementById('mediaModal');
const modalMedia = document.getElementById('modalMedia');

// Init Lucide
lucide.createIcons();

// Load data from LocalStorage
function initApp() {
  document.getElementById('arkKey').value = localStorage.getItem('ark_api_key') || '';
  document.getElementById('bochaKey').value = localStorage.getItem('bocha_api_key') || '';

  try {
    const savedMsgs = JSON.parse(localStorage.getItem('agent_messages') || '[]');
    messages = savedMsgs;
    if (messages.length > 0) {
      renderMessages();
    }
  } catch (e) {
    console.error("Failed to load history", e);
    messages = [];
  }

  autoResizeTextarea();
}

function saveKeys() {
  const ark = document.getElementById('arkKey').value.trim();
  const bocha = document.getElementById('bochaKey').value.trim();
  localStorage.setItem('ark_api_key', ark);
  localStorage.setItem('bocha_api_key', bocha);
  showToast("配置已更新");
}

function clearHistory() {
  if (confirm("确定要清空所有聊天历史吗？")) {
    messages = [];
    sessionId = '';
    localStorage.removeItem('agent_messages');
    localStorage.removeItem('agent_session_id');
    renderMessages();
    showToast("历史已清空");
  }
}

function toggleSidebar() {
  const sidebar = document.getElementById('sidebar');
  const overlay = document.getElementById('overlay');

  if (window.innerWidth > 768) {
    sidebar.classList.toggle('collapsed');
  } else {
    sidebar.classList.toggle('show');
    overlay.classList.toggle('show');
  }
}

function showToast(msg) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.classList.add('show');
  setTimeout(() => t.classList.remove('show'), 2000);
}

function autoResizeTextarea() {
  userInput.addEventListener('input', function () {
    this.style.height = '40px';
    this.style.height = (this.scrollHeight) + 'px';
  });
  userInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  });
}

function useSuggest(text) {
  userInput.value = text;
  userInput.focus();
  userInput.dispatchEvent(new Event('input'));
}

async function sendMessage() {
  const text = userInput.value.trim();
  if (!text || isBusy) return;

  const arkKey = localStorage.getItem('ark_api_key');
  if (!arkKey) {
    alert("请先在侧边栏配置 火山引擎 API Key");
    return;
  }

  isBusy = true;
  sendBtn.disabled = true;
  inputCard.classList.add('disabled');
  userInput.value = '';
  userInput.style.height = '40px';

  const userMsg = { role: 'user', content: text };
  messages.push(userMsg);
  renderMessages();

  try {
    const bochaKey = localStorage.getItem('bocha_api_key') || '';
    const response = await fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message: text,
        api_key: arkKey,
        bocha_key: bochaKey,
        session_id: sessionId
      })
    });

    if (!response.ok) throw new Error("网络请求失败");

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop();

      for (const part of parts) {
        const lines = part.split('\n');
        let event = '', data = '';
        for (const line of lines) {
          if (line.startsWith('event: ')) event = line.replace('event: ', '');
          else if (line.startsWith('data: ')) data = line.replace('data: ', '');
        }

        if (event === 'session') {
          sessionId = data;
          localStorage.setItem('agent_session_id', sessionId);
        } else if (event === 'message') {
          const msg = JSON.parse(data);
          const existingIdx = messages.findIndex(m => m.id === msg.id);
          if (existingIdx !== -1) messages[existingIdx] = msg;
          else messages.push(msg);
          renderMessages();
        } else if (event === 'error') {
          const err = JSON.parse(data);
          messages.push({ role: 'assistant', content: `**服务错误**: ${err.error}` });
          renderMessages();
        } else if (event === 'done') {
          finishResponse();
        }
      }
    }
    localStorage.setItem('agent_messages', JSON.stringify(messages.slice(-100)));
  } catch (err) {
    messages.push({ role: 'assistant', content: `**连接错误**: ${err.message}` });
    renderMessages();
  } finally {
    finishResponse();
  }
}

function finishResponse() {
  const loader = document.getElementById('tempLoading');
  if (loader) loader.remove();
  isBusy = false;
  sendBtn.disabled = false;
  inputCard.classList.remove('disabled');
  renderMessages();
}

function renderMessages() {
  chatList.innerHTML = '';
  if (messages.length === 0) {
    chatList.appendChild(welcomeScreen);
    welcomeScreen.style.display = 'flex';
    lucide.createIcons();
    return;
  }
  welcomeScreen.style.display = 'none';

  let i = 0;
  while (i < messages.length) {
    const m = messages[i];
    if (m.role === 'system') { i++; continue; }

    const msgDiv = document.createElement('div');
    msgDiv.className = `msg ${m.role === 'user' ? 'user' : 'ai'}`;
    const avatar = document.createElement('div');
    avatar.className = 'avatar';
    avatar.innerHTML = m.role === 'user' ? '我' : '<i data-lucide="bot" style="width:18px"></i>';

    const bubble = document.createElement('div');
    bubble.className = 'bubble';

    if (m.role === 'user') {
      bubble.textContent = m.content;
    } else if (m.role === 'assistant') {
      const filteredContent = filterRedundantURLs(m.content || "");
      if (!filteredContent.trim()) { i++; continue; }
      const mdContainer = document.createElement('div');
      mdContainer.className = 'markdown-body';
      mdContainer.innerHTML = marked.parse(filteredContent);
      bubble.appendChild(mdContainer);
      if (m.tool_calls && m.tool_calls.length) console.log('AI Tool Calls:', m.tool_calls);
    } else if (m.role === 'tool') {
      console.log(`Tool Result [${m.name}]:`, m.content);
      const mediaEl = renderMediaOnly(m.content, m.name);
      if (mediaEl) bubble.appendChild(mediaEl);
      else { i++; continue; }
    }

    msgDiv.appendChild(avatar);
    msgDiv.appendChild(bubble);
    chatList.appendChild(msgDiv);
    i++;
  }

  if (isBusy && !document.getElementById('tempLoading')) appendLoading();
  lucide.createIcons();
  chatView.scrollTop = chatView.scrollHeight;
}

function filterRedundantURLs(content) {
  if (!content) return "";
  // Simple regex to hide common media links if they are the only thing or part of a small text
  // Since we want to hide them anyway as per user rule
  return content.replace(/https?:\/\/[^\s)]+(?:\.jpg|\.png|\.gif|\.mp4|\.webm)[^\s)]*/gi, '').trim();
}

function renderMediaOnly(content, toolName) {
  if (toolName !== 'generate_image' && toolName !== 'generate_video' && toolName !== 'query_video_task') return null;
  try {
    const data = JSON.parse(content);
    const container = document.createElement('div');
    container.style = "margin-top: 8px;";

    if (data.image_url) {
      const media = document.createElement('div');
      media.className = 'media-container';
      media.onclick = () => showPreview(data.image_url, 'image');
      media.innerHTML = `<img src="${data.image_url}" loading="lazy">`;
      container.appendChild(media);
      container.appendChild(renderActions(data.image_url, 'ai-image.jpg'));
      return container;
    }

    if (data.video_url) {
      const media = document.createElement('div');
      media.className = 'media-container';
      media.onclick = () => showPreview(data.video_url, 'video');
      media.innerHTML = `<video src="${data.video_url}" muted loop onmouseover="this.play()" onmouseout="this.pause()"></video>`;
      container.appendChild(media);
      container.appendChild(renderActions(data.video_url, 'ai-video.mp4'));
      return container;
    }
  } catch (e) {
    console.error("Parse media error", e);
  }
  return null;
}

function renderActions(url, filename) {
  const actions = document.createElement('div');
  actions.className = 'media-actions';
  actions.innerHTML = `
    <button class="action-btn" onclick="event.stopPropagation(); copyToClipboard('${url}')"><i data-lucide="copy" style="width:12px"></i> 复制</button>
    <a href="${url}" download="${filename}" class="action-btn" target="_blank" onclick="event.stopPropagation()"><i data-lucide="download" style="width:12px"></i> 下载</a>
  `;
  return actions;
}

function showPreview(url, type) {
  modalMedia.innerHTML = type === 'image'
    ? `<img src="${url}" class="modal-content">`
    : `<video src="${url}" class="modal-content" controls autoplay></video>`;
  modal.classList.add('show');
}

function closePreview() {
  modal.classList.remove('show');
  modalMedia.innerHTML = '';
}

function appendLoading() {
  const msgDiv = document.createElement('div');
  msgDiv.className = 'msg ai';
  msgDiv.id = 'tempLoading';
  const avatar = document.createElement('div');
  avatar.className = 'avatar';
  avatar.innerHTML = '<i data-lucide="bot" style="width:18px"></i>';
  const bubble = document.createElement('div');
  bubble.className = 'bubble';
  bubble.innerHTML = '<div class="loading-pulse"><div class="dot"></div><div class="dot"></div><div class="dot"></div><span style="font-size:0.75rem; margin-left:6px; color:#94a3b8">思考中...</span></div>';
  msgDiv.appendChild(avatar);
  msgDiv.appendChild(bubble);
  chatList.appendChild(msgDiv);
  lucide.createIcons();
  chatView.scrollTop = chatView.scrollHeight;
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => showToast("已复制到剪贴板")).catch(err => console.error('Copy failed', err));
}

initApp();
