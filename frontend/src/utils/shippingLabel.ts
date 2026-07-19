// Shipping label rendering — shared by OrdersView (single + batch) and
// CustomersView. One place for the design, the sender address, and the
// print-window plumbing. Layout follows the owner's reference artwork:
// cream paper, gold double border, serif Latin, LINE Seed Thai, a real
// Code 128 barcode of the order id, product table and thank-you footer.

export interface LabelItem {
  name: string
  detail: string // e.g. "Size: S · Color: White"
  quantity: number
}

export interface LabelData {
  orderId?: number
  items?: LabelItem[]
  customer: { name?: string; phone?: string; address?: string }
}

// ---- palette (kept local: print HTML can't use app CSS vars) ----
const INK = '#2B2118'
const GOLD = '#A5824C'
const CREAM = '#FBF7EF'
const CARD = '#F3ECDF'
const LINE = '#DDD0B8'

const SENDER = {
  name: 'Bruno Collective',
  address1: '87/4-5 ถนน กลางเมือง ตำบลในเมือง',
  address2: 'อำเภอเมืองขอนแก่น ขอนแก่น 40000',
  tel: '095-296-4145',
}

const SOCIALS = [
  { icon: 'ig', text: 'brunocollective.th' },
  { icon: 'line', text: '@brunocollective' },
  { icon: 'fb', text: 'Bruno Collective' },
]

function esc(s: unknown): string {
  return String(s ?? '').replace(/[&<>"']/g, c => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string
  ))
}

// Format a Thai mobile number 0952964145 -> 095-296-4145 (leave others as-is).
function formatPhone(p?: string): string {
  const d = (p || '').replace(/\D/g, '')
  if (d.length === 10) return `${d.slice(0, 3)}-${d.slice(3, 6)}-${d.slice(6)}`
  return p || ''
}

// ---- Code 128 (set B) — encodes the order id into a real scannable barcode ----
const C128 = ('212222 222122 222221 121223 121322 131222 122213 122312 132212 221213 ' +
  '221312 231212 112232 122132 122231 113222 123122 123221 223211 221132 ' +
  '221231 213212 223112 312131 311222 321122 321221 312212 322112 322211 ' +
  '212123 212321 232121 111323 131123 131321 112313 132113 132311 211313 ' +
  '231113 231311 112133 112331 132131 113123 113321 133121 313121 211331 ' +
  '231131 213113 213311 213131 311123 311321 331121 312113 312311 332111 ' +
  '314111 221411 431111 111224 111422 121124 121421 141122 141221 112214 ' +
  '112412 122114 122411 142112 142211 241211 221114 413111 241112 134111 ' +
  '111242 121142 121241 114212 124112 124211 411212 421112 421211 212141 ' +
  '214121 412121 111143 111341 131141 114113 114311 411113 411311 113141 ' +
  '114131 311141 411131 211412 211214 211232').split(' ')
const C128_STOP = '2331112'

function code128Svg(text: string, heightMm = 11): string {
  // Code 128 set B; checksum = (start + Σ value_i × position_i) mod 103.
  const values = [104, ...[...text].map(ch => ch.charCodeAt(0) - 32)]
  let ck = values[0]
  for (let i = 1; i < values.length; i++) ck += values[i] * i
  const patterns = [...values, ck % 103].map(v => C128[v]).concat(C128_STOP)

  const unit = 0.33 // mm per narrow module
  let x = 0
  let rects = ''
  for (const pat of patterns) {
    for (let i = 0; i < pat.length; i++) {
      const w = Number(pat[i]) * unit
      if (i % 2 === 0) rects += `<rect x="${x.toFixed(2)}" y="0" width="${w.toFixed(2)}" height="${heightMm}" fill="${INK}"/>`
      x += w
    }
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${x.toFixed(2)}mm" height="${heightMm}mm" viewBox="0 0 ${x.toFixed(2)} ${heightMm}" preserveAspectRatio="xMidYMid meet" style="max-width:100%">${rects}</svg>`
}

// ---- tiny inline icons (stroke = gold) ----
const ICONS: Record<string, string> = {
  pin: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="${GOLD}" stroke-width="1.6"><path d="M12 21s-7-5.7-7-11a7 7 0 1 1 14 0c0 5.3-7 11-7 11z"/><circle cx="12" cy="10" r="2.6"/></svg>`,
  gift: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="${GOLD}" stroke-width="1.5"><rect x="3" y="8" width="18" height="4"/><rect x="5" y="12" width="14" height="9"/><path d="M12 8v13M12 8s-4 0-5-2c-.8-1.6.5-3 2-3 2.2 0 3 5 3 5zM12 8s4 0 5-2c.8-1.6-.5-3-2-3-2.2 0-3 5-3 5z"/></svg>`,
  ig: `<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="${INK}" stroke-width="2"><rect x="2.5" y="2.5" width="19" height="19" rx="5"/><circle cx="12" cy="12" r="4.5"/><circle cx="17.8" cy="6.2" r="1.4" fill="${INK}" stroke="none"/></svg>`,
  line: `<svg width="10" height="10" viewBox="0 0 24 24" fill="${INK}"><path d="M12 3C6.5 3 2 6.6 2 11.1c0 4 3.6 7.4 8.4 8 .3.1.8.2.9.5.1.3.1.7 0 1l-.1.9c0 .3-.2 1 .9.6 1.1-.5 6-3.6 8.2-6.1 1.5-1.7 2.2-3.4 2.2-5C22.5 6.6 17.5 3 12 3z"/></svg>`,
  fb: `<svg width="10" height="10" viewBox="0 0 24 24" fill="${INK}"><path d="M22 12a10 10 0 1 0-11.6 9.9v-7H7.9V12h2.5V9.8c0-2.5 1.5-3.9 3.8-3.9 1.1 0 2.2.2 2.2.2v2.5h-1.3c-1.2 0-1.6.8-1.6 1.6V12h2.8l-.4 2.9h-2.4v7A10 10 0 0 0 22 12z"/></svg>`,
}

// Render one label as an HTML string (used inside the print window).
export function renderLabel(data: LabelData): string {
  const c = data.customer || {}
  const hasOrder = data.orderId != null
  const items = data.items || []

  const orderBlock = hasOrder
    ? `<div class="lb-order">
         <div class="lb-order-no"><span>ORDER</span> #${esc(data.orderId)}</div>
         <div class="lb-barcode">${code128Svg(String(data.orderId))}</div>
         <div class="lb-itemcount">${items.reduce((n, i) => n + (i.quantity || 0), 0) || items.length || 1} ITEM(S)</div>
       </div>`
    : ''

  const productRows = items.map(i => `
      <tr>
        <td class="lb-td-item"><div class="lb-item-name">${esc(i.name)}</div></td>
        <td class="lb-td-detail">${esc(i.detail || '—')}</td>
        <td class="lb-td-qty">${esc(i.quantity)}</td>
      </tr>`).join('')

  const productTable = items.length
    ? `<div class="lb-sec-title">PRODUCT DETAILS</div>
       <table class="lb-table">
         <thead><tr><th>ITEM</th><th>DETAILS</th><th class="lb-th-qty">QTY</th></tr></thead>
         <tbody>${productRows}</tbody>
       </table>`
    : ''

  return `
  <div class="lb">
    <div class="lb-in">
      <div class="lb-head">
        <div class="lb-monogram">B<span>C</span></div>
        <div class="lb-brand">BRUNO&nbsp;COLLECTIVE</div>
        <div class="lb-tagline"><i></i><b>◇</b><i></i></div>
        <div class="lb-tag">TIMELESS ESSENTIALS</div>
      </div>

      <div class="lb-fromrow">
        <div class="lb-from">
          <div class="lb-sec-title">FROM (SENDER)</div>
          <div class="lb-from-name">${esc(SENDER.name)}</div>
          <div class="lb-from-detail">${esc(SENDER.address1)}</div>
          <div class="lb-from-detail">${esc(SENDER.address2)}</div>
          <div class="lb-from-detail">Tel: ${esc(SENDER.tel)}</div>
        </div>
        ${orderBlock}
      </div>

      <div class="lb-to">
        <div class="lb-to-toprow">
          <div class="lb-sec-title">TO (RECIPIENT)</div>
          <div class="lb-pin">${ICONS.pin}<span>DELIVERY<br>ADDRESS</span></div>
        </div>
        <div class="lb-to-name">${esc(c.name || '')}</div>
        ${c.phone ? `<div class="lb-to-phone">Tel: ${esc(formatPhone(c.phone))}</div>` : ''}
        <div class="lb-to-address">${esc(c.address || '')}</div>
      </div>

      ${productTable}

      <div class="lb-foot">
        <div class="lb-foot-thanks">
          ${ICONS.gift}
          <div><div class="lb-thanks-en">THANK YOU</div>
          <div class="lb-thanks-th">ขอบคุณที่อุดหนุนกับเรา ♡</div></div>
        </div>
        <div class="lb-foot-note">หากมีข้อสงสัยหรือสินค้ามีปัญหา<br>กรุณาติดต่อเราทันที</div>
        <div class="lb-foot-social">
          ${SOCIALS.map(s => `<div>${ICONS[s.icon]}<span>${esc(s.text)}</span></div>`).join('')}
        </div>
      </div>
    </div>
    <div class="lb-strip">CRAFTED FOR QUALITY.&nbsp;&nbsp;DESIGNED TO LAST.</div>
  </div>`
}

export const LABEL_CSS = `
  * { margin:0; padding:0; box-sizing:border-box; }
  @page { size: 100mm 150mm; margin: 0; }
  /* Thai glyphs in LINE Seed (served by the storefront at /fonts); Latin in serif. */
  @font-face { font-family:'LINE Seed Sans TH'; src:url('/fonts/LINESeedSansTH_W_Rg.woff2') format('woff2');
    font-weight:400; unicode-range:U+0E00-0E7F; }
  @font-face { font-family:'LINE Seed Sans TH'; src:url('/fonts/LINESeedSansTH_W_Bd.woff2') format('woff2');
    font-weight:500 700; unicode-range:U+0E00-0E7F; }
  @font-face { font-family:'LINE Seed Sans TH'; src:url('/fonts/LINESeedSansTH_W_XBd.woff2') format('woff2');
    font-weight:800 900; unicode-range:U+0E00-0E7F; }
  body { background:#fff; font-family:'LINE Seed Sans TH', Georgia, 'Times New Roman', serif; color:${INK}; }

  .lb { width:100mm; height:150mm; background:${CREAM}; border:0.5mm solid ${GOLD};
        border-radius:4.5mm; padding:1.4mm; display:flex; flex-direction:column; overflow:hidden;
        page-break-after:always; margin:0 auto; }
  .lb-in { border:0.25mm solid ${LINE}; border-radius:3.4mm; padding:4.5mm 5mm 3mm; flex:1;
           display:flex; flex-direction:column; min-height:0; }

  .lb-head { text-align:center; }
  .lb-monogram { font-size:17pt; color:${GOLD}; letter-spacing:-1pt; line-height:1; }
  .lb-monogram span { margin-left:-2pt; }
  .lb-brand { font-size:15.5pt; letter-spacing:3.5pt; margin-top:1mm; color:${INK}; }
  .lb-tagline { display:flex; align-items:center; gap:2mm; margin:1.2mm 8mm 0.8mm; color:${GOLD}; font-size:5pt; }
  .lb-tagline i { flex:1; border-top:0.2mm solid ${GOLD}; opacity:.6; }
  .lb-tag { font-size:5.5pt; letter-spacing:2.4pt; color:${GOLD}; }

  .lb-sec-title { font-size:5.5pt; letter-spacing:1.6pt; color:${GOLD}; font-weight:600; margin-bottom:1.4mm; }

  .lb-fromrow { display:flex; gap:4mm; border-top:0.2mm solid ${LINE}; margin-top:2.2mm; padding-top:2.2mm; }
  .lb-from { flex:1.15; }
  .lb-from-name { font-size:11pt; font-weight:700; margin-bottom:0.8mm; }
  .lb-from-detail { font-size:7.5pt; line-height:1.5; }
  .lb-order { flex:1; border-left:0.2mm solid ${LINE}; padding-left:4mm; text-align:center; }
  .lb-order-no { font-size:10pt; font-weight:700; margin-bottom:1.6mm; text-align:left; }
  .lb-order-no span { font-size:6.5pt; letter-spacing:1.4pt; color:${GOLD}; font-weight:600; margin-right:1.5mm; }
  .lb-barcode { line-height:0; }
  .lb-itemcount { font-size:6pt; letter-spacing:1.6pt; margin-top:1.4mm; color:${INK}; }

  .lb-to { background:${CARD}; border-radius:2.6mm; padding:3.2mm 4mm 3.6mm; margin-top:3mm; }
  .lb-to-toprow { display:flex; justify-content:space-between; align-items:flex-start; }
  .lb-pin { display:flex; align-items:center; gap:1.5mm; font-size:4.8pt; letter-spacing:1pt; color:${GOLD};
            text-align:right; font-weight:600; }
  .lb-to-name { font-size:16pt; font-weight:800; line-height:1.25; }
  .lb-to-phone { font-size:9.5pt; margin-top:1mm; }
  .lb-to-address { font-size:9.5pt; line-height:1.55; margin-top:1mm; white-space:pre-wrap; }

  .lb-table { width:100%; border-collapse:collapse; font-size:7.5pt; }
  .lb-table th { font-size:5.5pt; letter-spacing:1.4pt; color:${INK}; font-weight:600; background:${CARD};
                 border:0.2mm solid ${LINE}; padding:1.4mm; }
  .lb-table td { border:0.2mm solid ${LINE}; padding:1.6mm 2mm; vertical-align:middle; }
  .lb-th-qty { width:12mm; }
  .lb-item-name { font-weight:700; font-size:8pt; }
  .lb-td-detail { font-size:8pt; }
  .lb-td-qty { text-align:center; font-size:9pt; }
  .lb > .lb-in > .lb-table, .lb-in .lb-sec-title + .lb-table { margin-top:0; }
  .lb-in > .lb-sec-title { margin-top:3mm; }

  .lb-foot { margin-top:auto; padding-top:2.4mm; border-top:0.2mm solid ${LINE};
             display:flex; align-items:center; gap:3mm; }
  .lb-foot-thanks { display:flex; align-items:center; gap:2mm; flex:1.1; }
  .lb-thanks-en { font-size:7pt; letter-spacing:1.2pt; color:${GOLD}; font-weight:700; white-space:nowrap; }
  .lb-thanks-th { font-size:6.5pt; margin-top:0.6mm; }
  .lb-foot-note { flex:1.2; font-size:6pt; line-height:1.6; border-left:0.2mm solid ${LINE};
                  border-right:0.2mm solid ${LINE}; padding:0 3mm; }
  .lb-foot-social { flex:1; font-size:6pt; }
  .lb-foot-social div { display:flex; align-items:center; gap:1.4mm; margin:0.7mm 0; }
  .lb-strip { text-align:center; font-size:5.5pt; letter-spacing:2pt; color:${INK};
              background:${CARD}; border-radius:0 0 3.2mm 3.2mm; padding:1.7mm 0; margin-top:1mm; }

  @media print { body { background:#fff; } .lb { border-color:${GOLD}; } }
`

// Open a print window containing the given labels (one per page) and trigger
// the browser's print dialog. Fonts are loaded before printing so Thai text
// renders in LINE Seed instead of the system fallback.
export function printLabels(labels: LabelData[], title: string): void {
  if (!labels.length) return
  const win = window.open('', '_blank', 'width=520,height=760')
  if (!win) return
  win.document.write(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>${esc(title)}</title>
<style>${LABEL_CSS}</style></head><body>
${labels.map(renderLabel).join('')}
<script>
  window.onload = function () {
    var go = function () { window.print(); window.onafterprint = function () { window.close(); }; };
    if (document.fonts && document.fonts.ready) { document.fonts.ready.then(go); } else { go(); }
  };
<\/script>
</body></html>`)
  win.document.close()
}

// Build LabelItem rows from an order's items (uses the size/color snapshot
// stored on each order line, with the product relation for the name).
export function orderLabelItems(order: any): LabelItem[] {
  return (order?.items || []).map((it: any) => {
    const parts: string[] = []
    if (it.size) parts.push(`Size: ${it.size}`)
    if (it.color) parts.push(`Color: ${it.color}`)
    return {
      name: it.product?.name || '-',
      detail: parts.join('  ·  ') || '—',
      quantity: it.quantity || 1,
    }
  })
}
