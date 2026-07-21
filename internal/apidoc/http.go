package apidoc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register exposes an offline API document viewer and the generated OpenAPI document.
func Register(engine *gin.Engine) {
	document := Build(engine.Routes())
	engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/docs/ui")
	})
	engine.GET("/docs/openapi.json", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, document)
	})
	engine.GET("/docs/ui", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, documentHTML)
	})
}

const documentHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>AI Video API 文档</title>
  <style>
    :root{color-scheme:light;--bg:#f5f7fb;--panel:#fff;--line:#e4e8f0;--text:#182235;--muted:#68758a;--get:#1677ff;--post:#18a058;--put:#d97706;--patch:#8b5cf6;--delete:#dc2626}
    *{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.55 system-ui,-apple-system,"Segoe UI","PingFang SC",sans-serif}
    header{position:sticky;top:0;z-index:5;padding:18px 24px;border-bottom:1px solid var(--line);background:rgba(255,255,255,.94);backdrop-filter:blur(10px)}
    .head{display:flex;align-items:center;justify-content:space-between;gap:18px;max-width:1500px;margin:auto}.title{font-size:20px;font-weight:700}.sub{color:var(--muted);font-size:12px}
    .tools{display:flex;gap:10px;align-items:center}input{height:38px;padding:0 12px;border:1px solid var(--line);border-radius:8px;background:#fff;outline:none}#search{width:min(340px,38vw)}
    a.button{height:38px;padding:8px 13px;border-radius:8px;background:#111827;color:#fff;text-decoration:none;white-space:nowrap}
    main{max-width:1500px;margin:22px auto;padding:0 24px}.stats{margin-bottom:14px;color:var(--muted)}.common{padding:14px;border:1px solid var(--line);border-radius:10px;background:var(--panel)}.common h3{margin:0 0 4px}
    section{margin:0 0 18px}.tag-title{margin:16px 0 9px;font-size:16px}.endpoint{margin:8px 0;border:1px solid var(--line);border-radius:10px;background:var(--panel);overflow:hidden}
    summary{display:grid;grid-template-columns:72px minmax(280px,1.2fr) minmax(180px,1fr);gap:12px;align-items:center;padding:12px 14px;cursor:pointer;list-style:none}summary::-webkit-details-marker{display:none}
    .method{padding:4px 8px;border-radius:5px;color:#fff;text-align:center;font-size:12px;font-weight:700}.GET{background:var(--get)}.POST{background:var(--post)}.PUT{background:var(--put)}.PATCH{background:var(--patch)}.DELETE{background:var(--delete)}
    .path{overflow-wrap:anywhere;font-family:ui-monospace,SFMono-Regular,Consolas,monospace}.summary{color:var(--muted)}.detail{padding:0 14px 15px;border-top:1px solid var(--line)}
    h4{margin:14px 0 6px}table{width:100%;border-collapse:collapse}th,td{padding:7px 9px;border:1px solid var(--line);text-align:left;vertical-align:top}th{background:#f8fafc}
    pre{max-height:420px;margin:6px 0;padding:12px;border-radius:8px;background:#111827;color:#d8e2f1;overflow:auto;font-size:12px}.empty{padding:50px;text-align:center;color:var(--muted)}
    @media(max-width:760px){header{padding:14px}.head{align-items:stretch;flex-direction:column}.tools{flex-wrap:wrap}#search{flex:1;width:auto}main{padding:0 12px}summary{grid-template-columns:62px 1fr}.summary{grid-column:2}}
  </style>
</head>
<body>
<header><div class="head"><div><div class="title">AI Video 客户端 API 文档</div><div class="sub">仅展示 /api 接口 · OpenAPI 3.0.3</div></div><div class="tools"><input id="search" placeholder="搜索路径、功能或分组"><a class="button" href="/docs/openapi.json" target="_blank">OpenAPI JSON</a></div></div></header>
<main><section class="common"><h3>API 公共请求信息</h3><h4>鉴权</h4><div id="authentication" class="sub">正在读取鉴权说明…</div><h4>公共请求参数</h4><div class="sub">以下 Header 信息由客户端统一携带，接口参数表中不再重复展示。</div><div id="common-parameters"></div></section><div id="stats" class="stats">正在读取接口…</div><div id="content"></div></main>
<script>
const methodOrder={GET:1,POST:2,PUT:3,PATCH:4,DELETE:5};let endpoints=[];
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
function schemaOf(op){const content=op.requestBody?.content||{};const body=content['application/json']||content['application/x-www-form-urlencoded']||content['multipart/form-data']||content['application/octet-stream'];return body?.schema}
function schemaSummary(s={}){let value=s.type||'';if(s.format)value+=' '+s.format;if(s.enum)value+=' enum: '+s.enum.join(', ');if(s.minLength!=null)value+=' minLength: '+s.minLength;if(s.maxLength!=null)value+=' maxLength: '+s.maxLength;if(s.minItems!=null)value+=' minItems: '+s.minItems;if(s.maxItems!=null)value+=' maxItems: '+s.maxItems;if(s.minimum!=null)value+=' minimum: '+s.minimum;if(s.maximum!=null)value+=' maximum: '+s.maximum;if(s.pattern)value+=' pattern: '+s.pattern;return value}
function renderParameters(items=[]){if(!items.length)return '<div class="sub">无</div>';return '<table><thead><tr><th>名称</th><th>位置</th><th>必填</th><th>类型/约束</th><th>中文说明</th></tr></thead><tbody>'+items.map(p=>'<tr><td><code>'+esc(p.name)+'</code></td><td>'+esc(p.in)+'</td><td>'+(p.required?'是':'否')+'</td><td><code>'+esc(schemaSummary(p.schema))+'</code></td><td>'+esc(p.description||p.schema?.description||'')+'</td></tr>').join('')+'</tbody></table>'}
function renderCommonParameters(doc){const items=doc.components?.['x-common-request-parameters']||Object.values(doc.components?.parameters||{});document.querySelector('#common-parameters').innerHTML=renderParameters(items);const auth=doc.components?.securitySchemes?.bearerAuth;document.querySelector('#authentication').textContent=auth?.description||'鉴权接口使用 Bearer JWT。'}
function render(){const q=document.querySelector('#search').value.trim().toLowerCase();const rows=endpoints.filter(e=>!q||e.search.includes(q));document.querySelector('#stats').textContent='共 '+endpoints.length+' 个接口，当前显示 '+rows.length+' 个';const groups={};rows.forEach(e=>(groups[e.tag]??=[]).push(e));const html=Object.keys(groups).sort().map(tag=>'<section><h3 class="tag-title">'+esc(tag)+'</h3>'+groups[tag].sort((a,b)=>a.path.localeCompare(b.path)||methodOrder[a.method]-methodOrder[b.method]).map(e=>'<details class="endpoint"><summary><span class="method '+e.method+'">'+e.method+'</span><span class="path">'+esc(e.path)+'</span><span class="summary">'+esc(e.op.summary)+'</span></summary><div class="detail"><h4>说明</h4><div>'+esc(e.op.description||'')+'</div><h4>鉴权</h4><div>'+(e.op.security?'Bearer JWT':'公开接口')+'</div><h4>参数</h4>'+renderParameters(e.op['x-request-parameters']||e.op.parameters)+'<h4>请求体</h4>'+(schemaOf(e.op)?'<pre>'+esc(JSON.stringify(schemaOf(e.op),null,2))+'</pre>':'<div class="sub">无</div>')+'<h4>响应</h4><pre>'+esc(JSON.stringify(e.op.responses,null,2))+'</pre></div></details>').join('')+'</section>').join('');document.querySelector('#content').innerHTML=html||'<div class="empty">没有匹配的接口</div>'}
fetch('/docs/openapi.json').then(r=>r.json()).then(doc=>{renderCommonParameters(doc);Object.entries(doc.paths||{}).forEach(([path,methods])=>Object.entries(methods).forEach(([method,op])=>{const tag=op.tags?.[0]||'其它';endpoints.push({path,method:method.toUpperCase(),op,tag,search:(path+' '+tag+' '+(op.summary||'')).toLowerCase()})}));render()}).catch(err=>{document.querySelector('#content').innerHTML='<div class="empty">文档读取失败：'+esc(err.message)+'</div>'});
document.querySelector('#search').addEventListener('input',render);
</script>
</body></html>`
