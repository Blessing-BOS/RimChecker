<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import AppIcon from './components/AppIcon.vue'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import * as service from '../bindings/rimchecker/internal/desktop/service'
import type { Report, Preview, Settings } from '../bindings/rimchecker/internal/desktop/models'
import type { DiagnosticGroup } from '../bindings/rimchecker/internal/logparse/models'
import './style.css'

const filename = ref(''), status = ref('等待选择日志'), error = ref(''), notice = ref('')
const parsing = ref(false), sending = ref(false), preparing = ref(false), saving = ref(false)
const choosing = ref(false), canceling = ref(false), aiExpanded = ref(false)
const report = ref<Report | null>(null), selected = ref<DiagnosticGroup | null>(null)
const filter = ref('ALL'), page = ref(1), diagnosis = ref(''), preview = ref<Preview | null>(null)
const showSettings = ref(false), discardConfirm = ref(false), hasKey = ref(false)
const draft = ref<Settings>({base_url:'',model:'',timeout_seconds:240}), saved = ref('')
const keyDraft = ref(''), promptDraft = ref(''), savedPrompt = ref('')
const dirty = computed(() => JSON.stringify(draft.value)!==saved.value || keyDraft.value!=='' || promptDraft.value!==savedPrompt.value)
const filtered = computed(() => (report.value?.diagnostics ?? []).filter(g=>filter.value==='ALL'||(filter.value==='ERROR'?g.event.level!=='WARNING':g.event.level==='WARNING')))
const rows = computed(()=>filtered.value.slice((page.value-1)*100,page.value*100))
const md = new MarkdownIt({ html:false, linkify:false, breaks:true })
const rendered = computed(()=>DOMPurify.sanitize(md.render(diagnosis.value),{ALLOWED_TAGS:['p','br','h1','h2','h3','h4','strong','em','pre','code','ul','ol','li','blockquote','hr','table','thead','tbody','tr','th','td'],ALLOWED_ATTR:[]}))
function fail(e:unknown){error.value=String(e)}
function clean(){error.value='';notice.value=''}
function changeFilter(){page.value=1;selected.value=filtered.value[0]??null}
async function selectFile(){clean();choosing.value=true;try{const name=await service.SelectLogFile();if(name){filename.value=name;report.value=null;selected.value=null;diagnosis.value='';aiExpanded.value=false;status.value='已选择，等待解析'}}catch(e){fail(e)}finally{choosing.value=false}}
async function parse(){clean();parsing.value=true;diagnosis.value='';aiExpanded.value=false;selected.value=null;report.value=null;status.value='正在解析…';try{report.value=await service.ParseLog();page.value=1;status.value='解析完成';selected.value=filtered.value[0]??null}catch(e){status.value='解析失败';fail(e)}finally{parsing.value=false}}
async function copy(text:string){try{await navigator.clipboard.writeText(text);notice.value='已复制到剪贴板。'}catch{error.value='复制失败，请在文本区域手动选择并复制。'}}
async function openSettings(){clean();try{const state=await service.GetAIConfig();draft.value={...state.settings};saved.value=JSON.stringify(draft.value);hasKey.value=state.has_key;keyDraft.value='';promptDraft.value=await service.GetPrompt();savedPrompt.value=promptDraft.value;showSettings.value=true;if(state.warning)notice.value=state.warning}catch(e){fail(e)}}
function closeSettings(){if(saving.value)return;if(dirty.value){discardConfirm.value=true;return}showSettings.value=false;keyDraft.value=''}
function discardSettings(){showSettings.value=false;discardConfirm.value=false;keyDraft.value='';promptDraft.value=''}
async function saveAPI(){clean();saving.value=true;try{await service.SaveAIConfig(draft.value);if(keyDraft.value!=='')await service.SetAPIKey(keyDraft.value);keyDraft.value='';const state=await service.GetAIConfig();hasKey.value=state.has_key;saved.value=JSON.stringify(draft.value);notice.value='API 配置已保存；密钥仅保留在内存。'}catch(e){fail(e)}finally{saving.value=false}}
async function clearKey(){try{await service.SetAPIKey('');hasKey.value=false;keyDraft.value='';notice.value='本次会话密钥已清除。'}catch(e){fail(e)}}
async function loadPrompt(){if(promptDraft.value!==savedPrompt.value){notice.value='存在未保存修改，请先保存或取消编辑。';return}try{promptDraft.value=await service.GetPrompt();savedPrompt.value=promptDraft.value}catch(e){fail(e)}}
async function savePrompt(){clean();saving.value=true;try{await service.SavePrompt(promptDraft.value);savedPrompt.value=promptDraft.value;notice.value='提示词已保存到用户配置目录。'}catch(e){fail(e)}finally{saving.value=false}}
async function resetPrompt(){try{promptDraft.value=await service.ResetPrompt();notice.value='默认提示词已载入草稿，点击保存后生效。'}catch(e){fail(e)}}
async function prepare(){clean();preparing.value=true;try{preview.value=await service.PreviewAIRequest()}catch(e){fail(e)}finally{preparing.value=false}}
async function dismissPreview(){try{await service.DiscardPreview();preview.value=null}catch(e){fail(e)}}
async function send(){const snap=preview.value;if(!snap)return;preview.value=null;clean();sending.value=true;canceling.value=false;aiExpanded.value=true;diagnosis.value='';try{diagnosis.value=await service.DiagnoseWithAI(snap.token,true)}catch(e){fail(e)}finally{sending.value=false;canceling.value=false}}
async function cancel(){canceling.value=true;try{await service.CancelDiagnosis()}catch(e){canceling.value=false;fail(e)}}

// Keep keyboard focus inside modal dialogs and return it to the invoking control.
let returnFocus: HTMLElement | null = null
watch(() => showSettings.value || !!preview.value, async open => {
  if(open) returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  await nextTick()
  if(open) document.querySelector<HTMLElement>('.modal')?.focus()
  else if(returnFocus?.isConnected && !returnFocus.matches(':disabled')) returnFocus.focus()
})
watch(discardConfirm, async value => {
  await nextTick()
  document.querySelector<HTMLElement>(value ? '[data-continue]' : '.settings')?.focus()
})
function dialogKeys(event: KeyboardEvent, close: () => unknown) {
  if(event.key === 'Escape') {
    event.preventDefault()
    if(discardConfirm.value) discardConfirm.value = false
    else void close()
  }
  if(event.key !== 'Tab') return
  const modal = event.currentTarget as HTMLElement
  const items = Array.from(modal.querySelectorAll<HTMLElement>('button, input, select, textarea, [tabindex="0"]'))
    .filter(el => !el.matches(':disabled') && !el.closest('[inert]') && el.getClientRects().length)
  const first = items[0], last = items[items.length-1]
  if(!first || !last) { event.preventDefault(); return }
  if(!items.includes(document.activeElement as HTMLElement)) { event.preventDefault(); (event.shiftKey ? last : first).focus() }
  else if(event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
  else if(!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
}
onMounted(async()=>{try{const state=await service.GetAIConfig();if(state.warning)notice.value=state.warning}catch(e){fail(e)}})
</script>

<template>
  <div class="shell" :inert="showSettings || !!preview">
    <header class="toolbar">
      <div class="brand"><span class="brandmark" aria-hidden="true">R</span><h1>RimChecker</h1></div>
      <div class="tools">
        <button @click="selectFile" :disabled="parsing || sending || preparing || choosing"><AppIcon name="file" />选择日志</button>
        <button @click="parse" :disabled="!filename || parsing || sending || preparing || choosing"><AppIcon name="refresh" />{{ parsing ? '解析中…' : report ? '重新解析' : '开始解析' }}</button>
        <button @click="openSettings" :disabled="sending || parsing || preparing || choosing"><AppIcon name="settings" />设置</button>
      </div>
    </header>
    <div class="filebar">
      <span class="filename" :title="filename">{{ filename || '尚未选择日志' }}</span>
      <span v-if="report" class="tag">{{ report.format }}</span>
      <span class="file-status" role="status"><span class="status-dot" :class="{ busy: parsing }" />{{ status }}</span>
    </div>
    <div v-if="error" class="alert error" role="alert"><span>{{ error }}</span><button class="icon-button" aria-label="关闭错误提示" @click="error = ''"><AppIcon name="close" /></button></div>
    <div v-if="notice" class="alert" role="status"><span>{{ notice }}</span><button class="icon-button" aria-label="关闭提示" @click="notice = ''"><AppIcon name="close" /></button></div>

    <main>
      <section v-if="!report" class="welcome">
        <AppIcon name="file" />
        <h2>{{ parsing ? '正在解析日志' : filename ? '日志已就绪' : '选择一份日志，开始检查' }}</h2>
        <p>{{ parsing ? '正在本机提取异常与调用堆栈，请稍候。' : filename ? '点击开始解析，查看异常和重复报错。' : '支持 Player.log 和文本日志。解析在本机完成。' }}</p>
        <button v-if="!filename" class="primary" @click="selectFile" :disabled="choosing"><AppIcon name="file" />选择日志文件</button>
        <button v-else class="primary" @click="parse" :disabled="parsing || choosing">{{ parsing ? '解析中…' : '开始解析' }}</button>
      </section>
      <template v-else>
        <section class="stats" aria-label="日志统计">
          <div><small>日志总行数</small><strong>{{ report.total_lines.toLocaleString() }}</strong></div>
          <div><small>Error 次数</small><strong class="red">{{ report.errors.toLocaleString() }}</strong></div>
          <div><small>Warning 次数</small><strong class="amber">{{ report.warnings.toLocaleString() }}</strong></div>
          <div><small>去重异常组</small><strong>{{ report.diagnostics?.length ?? 0 }}</strong></div>
        </section>
        <section v-if="!report.diagnostics?.length" class="welcome panel">
          <AppIcon name="check" /><h2>{{ report.total_lines === 0 ? '日志文件为空' : '未发现可识别的异常' }}</h2>
          <p>{{ report.total_lines === 0 ? '请选择有内容的日志文件。' : '当前解析规则未发现异常，不代表日志中不存在问题。' }}</p>
        </section>
        <section v-else class="analysis">
          <div class="list panel">
            <div class="sectionhead"><h2>异常记录 <span class="muted">{{ filtered.length }}</span></h2><select v-model="filter" aria-label="筛选异常等级" @change="changeFilter"><option value="ALL">全部等级</option><option value="ERROR">Error / Fatal</option><option value="WARNING">Warning</option></select></div>
            <div :key="filter + page" class="records" aria-label="异常列表">
              <div v-if="!rows.length" class="empty">当前筛选下没有记录</div>
              <button v-for="g in rows" :key="g.first_start_line" class="record" :class="{ active: selected?.first_start_line === g.first_start_line }" :aria-pressed="selected?.first_start_line === g.first_start_line" @click="selected = g">
                <span class="record-heading"><span class="level" :class="g.event.level === 'WARNING' ? 'amber' : 'red'">{{ g.event.level }}</span><span class="record-title" :title="g.event.title || g.event.message">{{ g.event.title || g.event.message || '无标题记录' }}</span><span class="count">×{{ g.count }}</span></span>
                <small>首次出现于第 {{ g.first_start_line }} 行</small>
              </button>
            </div>
            <div v-if="filtered.length > 100" class="pager"><button :disabled="page === 1" @click="page--">上一页</button><span>{{ page }} / {{ Math.ceil(filtered.length / 100) }}</span><button :disabled="page * 100 >= filtered.length" @click="page++">下一页</button></div>
          </div>
          <div class="detail panel">
            <div class="sectionhead"><h2>异常详情</h2><span class="muted">原始日志证据</span></div>
            <div v-if="!selected" class="empty">当前筛选下没有可显示的异常</div>
            <div v-else :key="selected.first_start_line" class="detailbody" tabindex="0" aria-label="异常详情内容">
              <div class="metadata"><span :class="selected.event.level === 'WARNING' ? 'amber' : 'red'">{{ selected.event.level }}</span><span>发生 {{ selected.count }} 次</span><span>首次 {{ selected.first_start_line }}–{{ selected.first_end_line }} 行</span><span>末次 {{ selected.last_start_line }}–{{ selected.last_end_line }} 行</span><span v-if="selected.event.source" class="source">来源 {{ selected.event.source }}</span></div>
              <p v-if="selected.event.boundary_uncertain" class="uncertain">事件边界不确定，保留的上下文需人工复核。</p>
              <h3>完整消息</h3><pre class="message">{{ selected.event.message }}</pre>
              <div class="stackhead"><h3>调用堆栈</h3><button @click="copy((selected.event.stack ?? []).join('\n'))" :disabled="!selected.event.stack?.length"><AppIcon name="copy" />复制堆栈</button></div>
              <pre v-if="selected.event.stack?.length" class="stack" tabindex="0" aria-label="调用堆栈">{{ selected.event.stack.join('\n') }}</pre>
              <p v-else class="muted">未提取到堆栈，记录可能没有堆栈或信息不完整。</p>
            </div>
          </div>
        </section>
      </template>

      <section class="panel ai" :class="{ expanded: aiExpanded }">
        <div class="sectionhead">
          <button class="ai-toggle" :aria-expanded="aiExpanded" aria-controls="ai-content" @click="aiExpanded = !aiExpanded"><AppIcon name="chevron" :class="{ rotated: aiExpanded }" /><span>AI 辅助诊断</span><span class="muted">{{ sending ? '正在分析…' : diagnosis ? '诊断已生成' : '预览后授权发送' }}</span></button>
          <div class="tools">
            <button v-if="diagnosis && !sending" @click="copy(diagnosis)"><AppIcon name="copy" />复制 Markdown</button>
            <button v-if="sending" class="danger" @click="cancel" :disabled="canceling">{{ canceling ? '正在取消…' : '取消请求' }}</button>
            <button v-else @click="prepare" :disabled="!report?.diagnostics?.length || parsing || preparing || choosing">{{ preparing ? '准备预览…' : '使用 AI 分析' }}</button>
          </div>
        </div>
        <div v-show="aiExpanded" id="ai-content" class="ai-content" tabindex="0" aria-label="AI 诊断内容">
          <p v-if="sending" class="ai-status" role="status"><span class="status-dot busy" />{{ canceling ? '正在取消本次请求…' : '正在等待 AI 服务响应。可以随时取消，不会自动重试。' }}</p>
          <template v-else-if="diagnosis"><p class="result-note">模型生成的建议，请结合日志人工核实。</p><article class="markdown" v-html="rendered" /></template>
          <p v-else class="empty">AI 将解释日志并提供排查建议。发送前可检查内容，只有确认后才会上传。</p>
        </div>
      </section>
    </main>
    <footer><span>本地解析 · 不修改游戏文件</span><span>AI 上传需明确授权</span></footer>
  </div>

  <div v-if="showSettings" class="overlay" @keydown="dialogKeys($event, closeSettings)">
    <section class="modal settings" role="dialog" aria-modal="true" aria-labelledby="settings-title" tabindex="-1">
      <div class="sectionhead"><h2 id="settings-title">设置</h2><button class="icon-button" aria-label="关闭设置" @click="closeSettings"><AppIcon name="close" /></button></div>
      <div class="modalcontent" :inert="discardConfirm">
        <div v-if="error" class="alert error" role="alert">{{ error }}</div><div v-if="notice" class="alert" role="status">{{ notice }}</div>
        <h3>API 配置</h3><p class="muted">配置保存在用户目录。API Key 仅在本次运行中保留。</p>
        <label>API Base URL<input v-model="draft.base_url" placeholder="https://api.example.com/v1" spellcheck="false" :disabled="saving" /></label>
        <div class="formrow"><label>模型名称<input v-model="draft.model" placeholder="服务商支持的模型" :disabled="saving" /></label><label>超时（秒）<input v-model.number="draft.timeout_seconds" type="number" min="1" max="600" :disabled="saving" /></label></div>
        <label>API Key（仅本次运行）<input v-model="keyDraft" type="password" autocomplete="off" :placeholder="hasKey ? '已设置；留空保留现有密钥' : '请输入 API Key'" :disabled="saving" /></label>
        <div class="actions"><button class="primary" @click="saveAPI" :disabled="saving">保存 API 配置</button><button @click="clearKey" :disabled="saving || !hasKey">清除会话密钥</button></div>
        <h3 class="promptheading">诊断提示词</h3><textarea v-model="promptDraft" rows="7" aria-label="诊断提示词" spellcheck="false" :disabled="saving" />
        <div class="actions"><button @click="loadPrompt" :disabled="saving">加载当前提示词</button><button @click="resetPrompt" :disabled="saving">恢复默认到草稿</button><button class="primary" @click="savePrompt" :disabled="saving">保存提示词</button></div>
      </div>
      <div v-if="discardConfirm" class="discard" role="alert"><p>放弃未保存的修改？</p><div class="actions"><button data-continue @click="discardConfirm = false">继续编辑</button><button class="danger" @click="discardSettings">放弃修改并关闭</button></div></div>
      <div v-else class="modalfooter"><span class="muted">{{ dirty ? '有未保存的修改' : '没有未保存的修改' }} · 提示词不得含密钥</span><button @click="closeSettings" :disabled="saving">完成</button></div>
    </section>
  </div>

  <div v-if="preview" class="overlay" @keydown="dialogKeys($event, dismissPreview)">
    <section class="modal" role="dialog" aria-modal="true" aria-labelledby="preview-title" tabindex="-1">
      <div class="sectionhead"><h2 id="preview-title">确认发送内容</h2><button class="icon-button" aria-label="关闭发送预览" @click="dismissPreview"><AppIcon name="close" /></button></div>
      <div class="modalcontent">
        <div v-if="error" class="alert error" role="alert">{{ error }}</div>
        <div class="alert">数据将发送至下方第三方 AI 服务。自动清理无法覆盖所有敏感信息，请检查完整内容。</div>
        <dl class="preview-target"><dt>接口</dt><dd>{{ preview.endpoint }}</dd><dt>模型</dt><dd>{{ preview.request.model }}</dd></dl>
        <div v-for="(m, i) in preview.request.messages" :key="i"><h3>{{ m.role === 'user' ? '日志分析输入（不可信数据）' : '系统提示词' }}</h3><pre class="previewtext">{{ m.role === 'user' ? JSON.stringify(JSON.parse(m.content), null, 2) : m.content }}</pre></div>
      </div>
      <div class="modalfooter"><span class="muted">仅发送本次预览内容</span><div class="actions"><button @click="dismissPreview">取消，不发送</button><button class="primary" @click="send">确认向此服务发送一次</button></div></div>
    </section>
  </div>
</template>
