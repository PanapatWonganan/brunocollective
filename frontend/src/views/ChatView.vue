<template>
  <div>
    <div class="d-flex align-center mb-4">
      <div>
        <div class="text-h5 font-weight-bold">Chats</div>
        <div class="text-caption text-medium-emphasis">
          แชทลูกค้าจากทุกช่องทางในที่เดียว
          <v-chip size="x-small" variant="tonal" :color="wsConnected ? 'success' : 'warning'" class="ml-1">
            {{ wsConnected ? 'realtime' : 'อัปเดตทุก 20 วิ' }}
          </v-chip>
        </div>
      </div>
      <v-spacer />
      <v-btn icon="mdi-refresh" variant="text" :loading="loading" @click="fetchConversations" />
    </div>

    <v-card>
      <div class="chat-shell">
        <!-- Conversation list -->
        <div class="conv-pane" :class="{ 'conv-pane--hidden': mobile && activeConv }">
          <div class="conv-toolbar pa-3 pb-0">
            <v-text-field
              v-model="search" density="compact" hide-details clearable
              placeholder="ค้นหาชื่อ / ข้อความ..." prepend-inner-icon="mdi-magnify" class="mb-2"
            />
            <div class="d-flex ga-1 mb-2 flex-wrap">
              <v-chip
                v-for="p in platformFilters" :key="p.value" size="small" label
                :variant="platformFilter === p.value ? 'flat' : 'tonal'"
                :color="platformFilter === p.value ? 'primary' : undefined"
                @click="platformFilter = p.value"
              >{{ p.label }}</v-chip>
            </div>
            <v-tabs v-model="statusTab" density="compact" color="secondary" grow>
              <v-tab value="waiting" class="text-none px-1">
                รอตอบ
                <v-chip v-if="waitingCount" size="x-small" color="error" variant="flat" class="ml-1">{{ waitingCount }}</v-chip>
              </v-tab>
              <v-tab value="active" class="text-none px-1">กำลังคุย</v-tab>
              <v-tab value="deals" class="text-none px-1">
                ดีลค้าง
                <v-chip v-if="dealsCount" size="x-small" color="warning" variant="flat" class="ml-1">{{ dealsCount }}</v-chip>
              </v-tab>
              <v-tab value="done" class="text-none px-1">จบแล้ว</v-tab>
            </v-tabs>
            <v-divider />
          </div>
          <div v-if="filteredConversations.length" class="conv-list">
            <div
              v-for="conv in filteredConversations" :key="conv.id"
              class="conv-item pa-3"
              :class="{ 'conv-item--active': activeConv?.id === conv.id }"
              @click="openConversation(conv)"
            >
              <v-badge
                :model-value="conv.unread_count > 0" :content="conv.unread_count"
                color="error" offset-x="6" offset-y="6"
              >
                <v-avatar size="42" color="grey-lighten-3">
                  <v-img v-if="conv.avatar_url" :src="conv.avatar_url" cover />
                  <span v-else class="text-body-2 font-weight-bold">{{ (conv.display_name || '?')[0] }}</span>
                </v-avatar>
              </v-badge>
              <div class="conv-item-body ml-3">
                <div class="d-flex align-center">
                  <v-icon :icon="platformIcon(conv.platform)" size="14" :color="platformColor(conv.platform)" class="mr-1" />
                  <span class="font-weight-medium text-body-2 text-truncate">{{ conv.display_name }}</span>
                  <v-spacer />
                  <span class="text-caption text-medium-emphasis">{{ timeAgo(conv.last_message_at) }}</span>
                </div>
                <div class="text-caption text-medium-emphasis text-truncate">{{ conv.last_message_text || '—' }}</div>
                <div v-if="waitingLabel(conv) || dealLabel(conv) || (conv.tags && conv.tags.length)" class="d-flex ga-1 mt-1 flex-wrap">
                  <v-chip
                    v-if="dealLabel(conv)" size="x-small" label variant="tonal"
                    color="warning" prepend-icon="mdi-cash-clock"
                  >{{ dealLabel(conv) }}</v-chip>
                  <v-chip
                    v-if="waitingLabel(conv)" size="x-small" label variant="tonal"
                    :color="waitingUrgent(conv) ? 'error' : 'warning'" prepend-icon="mdi-clock-outline"
                  >{{ waitingLabel(conv) }}</v-chip>
                  <v-chip v-for="tag in conv.tags || []" :key="tag" size="x-small" label variant="tonal" color="secondary">{{ tag }}</v-chip>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-center pa-10 text-medium-emphasis">
            <template v-if="conversations.length">
              <v-icon :icon="statusTab === 'waiting' ? 'mdi-check-all' : 'mdi-filter-outline'" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">{{ statusTab === 'waiting' ? 'ตอบครบทุกคนแล้ว 🎉' : 'ไม่มีแชทในหมวดนี้' }}</div>
            </template>
            <template v-else>
              <v-icon icon="mdi-chat-outline" size="40" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2 mb-1">ยังไม่มีแชทเข้ามา</div>
              <div class="text-caption" style="max-width: 260px; margin: 0 auto;">
                ตั้งค่า webhook ของ LINE / Facebook / Instagram<br>มาที่ <code>{{ webhookURL }}</code>
              </div>
            </template>
          </div>
        </div>

        <!-- Thread -->
        <div class="thread-pane" :class="{ 'thread-pane--hidden': mobile && !activeConv }">
          <template v-if="activeConv">
            <!-- Thread header -->
            <div class="thread-header pa-3 d-flex align-center">
              <v-btn v-if="mobile" icon="mdi-arrow-left" size="small" variant="text" class="mr-1" @click="activeConv = null" />
              <v-avatar size="36" color="grey-lighten-3" class="mr-2">
                <v-img v-if="activeConv.avatar_url" :src="activeConv.avatar_url" cover />
                <span v-else class="text-body-2 font-weight-bold">{{ (activeConv.display_name || '?')[0] }}</span>
              </v-avatar>
              <div>
                <div class="font-weight-medium text-body-2">{{ activeConv.display_name }}</div>
                <div class="text-caption text-medium-emphasis d-flex align-center">
                  <v-icon :icon="platformIcon(activeConv.platform)" size="12" :color="platformColor(activeConv.platform)" class="mr-1" />
                  {{ activeConv.platform.toUpperCase() }}
                  <template v-if="activeConv.customer">
                    · ลูกค้า: {{ activeConv.customer.name }}
                  </template>
                </div>
              </div>
              <v-spacer />
              <div class="d-flex ga-1 align-center flex-wrap justify-end">
                <v-chip v-for="tag in activeConv.tags || []" :key="tag" size="x-small" label variant="tonal" color="secondary">{{ tag }}</v-chip>
                <v-btn icon="mdi-tag-outline" size="small" variant="text" title="ป้ายกำกับ" @click="openTagDialog" />
                <v-btn
                  v-if="aiEnabled"
                  size="small" variant="text" class="text-none"
                  :prepend-icon="activeConv.ai_disabled ? 'mdi-robot-off-outline' : 'mdi-robot-outline'"
                  :color="activeConv.ai_disabled ? undefined : 'success'"
                  :loading="aiToggling"
                  :title="activeConv.ai_disabled ? 'AI ปิดอยู่ในแชทนี้ — กดเพื่อเปิด' : 'AI ตอบแชทนี้อัตโนมัติ — กดเพื่อปิด'"
                  @click="toggleAI"
                >{{ activeConv.ai_disabled ? 'AI ปิด' : 'AI เปิด' }}</v-btn>
                <v-btn
                  size="small" variant="tonal" color="primary" class="text-none"
                  prepend-icon="mdi-cart-plus"
                  @click="openOrderDialog"
                >สร้างออเดอร์</v-btn>
                <v-btn
                  size="small" variant="text" class="text-none"
                  prepend-icon="mdi-account-link-outline"
                  @click="openLinkDialog"
                >
                  {{ activeConv.customer ? 'เปลี่ยนลูกค้า' : 'ผูกลูกค้า' }}
                </v-btn>
                <v-btn
                  v-if="isWaiting(activeConv)"
                  size="small" variant="tonal" color="warning" class="text-none"
                  prepend-icon="mdi-message-check-outline" :loading="answeredSaving"
                  title="ตอบลูกค้าจากแอป LINE OA ไปแล้ว — กดเพื่อเคลียร์สถานะรอตอบ (LINE ไม่ส่งข้อความที่ตอบจากแอปมาให้ระบบ)"
                  @click="markAnswered"
                >ตอบจากที่อื่นแล้ว</v-btn>
                <v-btn
                  v-if="activeConv.status !== 'done'"
                  size="small" variant="tonal" color="success" class="text-none"
                  prepend-icon="mdi-check" :loading="statusSaving" @click="toggleStatus('done')"
                >จบงาน</v-btn>
                <v-btn
                  v-else
                  size="small" variant="tonal" color="warning" class="text-none"
                  prepend-icon="mdi-restore" :loading="statusSaving" @click="toggleStatus('open')"
                >เปิดใหม่</v-btn>
              </div>
            </div>
            <v-divider />

            <!-- Unpaid chat orders (ดีลค้าง) for this thread -->
            <v-alert
              v-if="activeDeals.length"
              type="warning" variant="tonal" density="compact" class="ma-3 mb-0"
            >
              <div
                v-for="d in activeDeals" :key="d.id"
                class="d-flex align-center flex-wrap ga-2"
              >
                <span class="text-body-2">
                  ออเดอร์ #{{ d.id }} ฿{{ fmtMoney(d.total_amount) }} — ยังไม่ได้รับสลิป ({{ hoursSince(d.created_at) }})
                </span>
                <v-spacer />
                <v-btn size="x-small" variant="tonal" class="text-none" @click="draftFollowup(d)">
                  ส่งข้อความตาม
                </v-btn>
                <v-btn
                  v-if="!d.coupon_code"
                  size="x-small" variant="tonal" color="secondary" class="text-none"
                  @click="openDiscountDialog(d)"
                >ตามพร้อมส่วนลด</v-btn>
              </div>
            </v-alert>

            <!-- Messages -->
            <div ref="threadEl" class="thread-messages pa-4">
              <div
                v-for="msg in messages" :key="msg.id"
                class="d-flex mb-2"
                :class="msg.direction === 'out' ? 'justify-end' : 'justify-start'"
              >
                <div class="bubble" :class="msg.direction === 'out' ? 'bubble--out' : 'bubble--in'">
                  <v-img
                    v-if="msg.type === 'image' && msg.image_url"
                    :src="msg.image_url" max-width="240" rounded="lg" class="mb-1" cover
                  />
                  <div v-if="msg.text" class="text-body-2" style="white-space: pre-wrap;">{{ msg.text }}</div>
                  <div class="bubble-time text-caption">
                    <span v-if="sourceLabel(msg.source)">{{ sourceLabel(msg.source) }} · </span>{{ formatTime(msg.created_at) }}
                  </div>
                </div>
              </div>
            </div>

            <!-- Composer -->
            <v-divider />
            <div class="pa-3 d-flex align-end ga-2">
              <v-menu location="top start">
                <template v-slot:activator="{ props }">
                  <v-btn v-bind="props" icon="mdi-flash-outline" variant="tonal" color="secondary" title="ข้อความสำเร็จรูป" />
                </template>
                <v-list density="compact" max-height="300" width="300">
                  <v-list-item
                    v-for="cr in cannedReplies" :key="cr.id"
                    :title="cr.title" :subtitle="cr.text"
                    @click="draft = draft ? draft + '\n' + cr.text : cr.text"
                  />
                  <v-list-item v-if="!cannedReplies.length" title="ยังไม่มีข้อความสำเร็จรูป" disabled />
                  <v-divider />
                  <v-list-item prepend-icon="mdi-cog-outline" title="จัดการข้อความสำเร็จรูป" @click="cannedDialog = true" />
                </v-list>
              </v-menu>
              <v-textarea
                v-model="draft" placeholder="พิมพ์ข้อความตอบกลับ..." rows="1" auto-grow max-rows="4"
                hide-details density="compact" @keydown.enter.exact.prevent="sendReply"
              />
              <v-btn
                color="primary" icon="mdi-send" :loading="sending"
                :disabled="!draft.trim()" @click="sendReply"
              />
            </div>
            <v-alert v-if="sendError" type="error" variant="tonal" density="compact" class="ma-3 mt-0">
              {{ sendError }}
            </v-alert>
          </template>
          <div v-else class="d-flex align-center justify-center fill-height text-medium-emphasis pa-10">
            <div class="text-center">
              <v-icon icon="mdi-forum-outline" size="48" class="mb-3" color="grey-lighten-1" />
              <div class="text-body-2">เลือกแชทจากรายการด้านซ้าย</div>
            </div>
          </div>
        </div>
      </div>
    </v-card>

    <!-- Tags dialog -->
    <v-dialog v-model="tagDialog" max-width="440">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ป้ายกำกับแชท</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-combobox
            v-model="tagDraft" :items="tagSuggestions" multiple chips closable-chips
            label="ป้ายกำกับ" hint="พิมพ์เองแล้วกด Enter เพื่อเพิ่มป้ายใหม่" persistent-hint
          />
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="tagDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="tagSaving" @click="saveTags">บันทึก</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Canned replies manager -->
    <v-dialog v-model="cannedDialog" max-width="560">
      <v-card>
        <v-card-title class="pa-5 pb-2 d-flex align-center">
          <span class="text-h6 font-weight-bold">ข้อความสำเร็จรูป</span>
          <v-spacer />
          <v-btn size="small" variant="tonal" color="primary" class="text-none" prepend-icon="mdi-plus"
            @click="editCanned(null)">เพิ่มใหม่</v-btn>
        </v-card-title>
        <v-card-text class="px-5 pb-5">
          <template v-if="cannedForm">
            <v-text-field v-model="cannedForm.title" label="ชื่อ (เช่น ค่าส่ง, เลขบัญชี)" class="mb-1" />
            <v-textarea v-model="cannedForm.text" label="ข้อความ" rows="3" />
            <div class="d-flex justify-end ga-2">
              <v-btn variant="text" class="text-none" @click="cannedForm = null">ยกเลิก</v-btn>
              <v-btn color="primary" class="text-none" :loading="cannedSaving"
                :disabled="!cannedForm.title.trim() || !cannedForm.text.trim()" @click="saveCanned">บันทึก</v-btn>
            </div>
          </template>
          <v-list v-else density="comfortable">
            <v-list-item v-for="cr in cannedReplies" :key="cr.id" :title="cr.title" :subtitle="cr.text">
              <template v-slot:append>
                <v-btn icon="mdi-pencil-outline" size="x-small" variant="text" @click="editCanned(cr)" />
                <v-btn icon="mdi-delete-outline" size="x-small" variant="text" color="error" @click="deleteCanned(cr)" />
              </template>
            </v-list-item>
            <div v-if="!cannedReplies.length" class="text-center pa-6 text-medium-emphasis text-body-2">
              ยังไม่มีข้อความสำเร็จรูป — เพิ่มคำตอบที่พิมพ์บ่อย เช่น ค่าส่ง เลขบัญชี ไซส์ชาร์ต
            </div>
          </v-list>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Create order from chat dialog -->
    <v-dialog v-model="orderDialog" max-width="640" persistent>
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">สร้างออเดอร์จากแชท</span>
        </v-card-title>
        <v-card-text class="px-5">
          <v-alert
            v-if="!activeConv?.customer"
            type="warning" variant="tonal" density="compact" class="mb-4"
          >
            ต้องผูกลูกค้ากับแชทนี้ก่อน จึงจะสร้างออเดอร์ได้
            <template v-slot:append>
              <v-btn size="small" variant="text" class="text-none" @click="openLinkDialog">ผูกลูกค้า</v-btn>
            </template>
          </v-alert>
          <div v-else class="d-flex align-center mb-4">
            <v-icon icon="mdi-account" size="18" class="mr-2" />
            <span class="text-body-2 font-weight-medium">{{ activeConv.customer.name }}</span>
            <v-chip v-if="orderCustomerIsMember" size="x-small" color="secondary" variant="tonal" label class="ml-2">
              Member −5%
            </v-chip>
          </div>

          <v-card v-for="(item, idx) in orderItems" :key="idx" variant="outlined" class="pa-3 mb-2">
            <div class="d-flex ga-2 align-center">
              <v-select
                v-model="item.product_id" :items="products" item-title="name" item-value="id"
                label="สินค้า" density="compact" hide-details class="flex-grow-1"
                @update:model-value="item.variant_id = null"
              />
              <v-text-field
                v-model.number="item.quantity" label="จำนวน" type="number" min="1"
                density="compact" hide-details style="max-width: 90px"
              />
              <v-btn
                icon="mdi-close" size="x-small" variant="text"
                :disabled="orderItems.length === 1" @click="orderItems.splice(idx, 1)"
              />
            </div>
            <v-select
              v-if="variantsFor(item.product_id).length"
              v-model="item.variant_id" :items="variantsFor(item.product_id)"
              :item-title="variantLabel" item-value="id" label="Size / Color"
              density="compact" hide-details class="mt-2"
            />
          </v-card>
          <v-btn
            size="small" variant="text" class="text-none mb-2" prepend-icon="mdi-plus"
            @click="orderItems.push({ product_id: 0, variant_id: null, quantity: 1 })"
          >เพิ่มสินค้า</v-btn>

          <div v-if="orderSuggestions.length" class="mb-4">
            <div class="text-caption text-medium-emphasis mb-1">
              <v-icon icon="mdi-lightbulb-on-outline" size="14" class="mr-1" />ลูกค้ามักซื้อคู่กับ:
            </div>
            <v-chip
              v-for="s in orderSuggestions" :key="s.id"
              size="small" variant="tonal" color="secondary" label class="mr-1 mb-1"
              prepend-icon="mdi-plus"
              @click="addSuggested(s)"
            >{{ s.name }} · ฿{{ fmtMoney(s.price) }}</v-chip>
          </div>

          <div class="d-flex ga-2 align-center mb-1">
            <v-text-field
              v-model="orderCoupon" label="โค้ดส่วนลด (ถ้ามี)" density="compact" hide-details
              :disabled="!!appliedOrderCoupon" @keydown.enter.prevent="applyOrderCoupon"
            />
            <v-btn
              v-if="!appliedOrderCoupon" size="small" variant="tonal" class="text-none"
              :loading="orderCouponChecking" :disabled="!orderCoupon.trim() || !orderSubtotal"
              @click="applyOrderCoupon"
            >ใช้โค้ด</v-btn>
            <v-btn v-else size="small" variant="text" class="text-none" @click="removeOrderCoupon">เอาออก</v-btn>
          </div>
          <div v-if="orderCouponError" class="text-caption text-error mb-2">{{ orderCouponError }}</div>
          <div v-if="appliedOrderCoupon" class="text-caption text-success mb-2">
            ใช้โค้ด {{ appliedOrderCoupon.code }} — ลด ฿{{ fmtMoney(orderCouponDiscount) }}
          </div>

          <v-card variant="tonal" class="pa-3 mb-3">
            <div class="d-flex justify-space-between text-body-2">
              <span>ยอดรวม</span><span>฿{{ fmtMoney(orderSubtotal) }}</span>
            </div>
            <div v-if="orderMemberDiscount" class="d-flex justify-space-between text-body-2">
              <span>ส่วนลดสมาชิก 5%</span><span>−฿{{ fmtMoney(orderMemberDiscount) }}</span>
            </div>
            <div v-if="orderCouponDiscount" class="d-flex justify-space-between text-body-2">
              <span>คูปอง {{ appliedOrderCoupon?.code }}</span><span>−฿{{ fmtMoney(orderCouponDiscount) }}</span>
            </div>
            <v-divider class="my-2" />
            <div class="d-flex justify-space-between font-weight-bold">
              <span>ยอดชำระ</span><span>฿{{ fmtMoney(orderTotal) }}</span>
            </div>
          </v-card>

          <v-textarea v-model="orderNotes" label="หมายเหตุ (ถ้ามี)" rows="2" density="compact" hide-details class="mb-3" />

          <v-checkbox
            v-model="orderSendLink" density="compact" hide-details class="mb-1"
            label="ส่งสรุปออเดอร์ + ลิงก์ชำระเงินเข้าแชททันที"
          />

          <v-alert v-if="orderError" type="error" variant="tonal" density="compact" class="mb-2">
            {{ orderError }}
          </v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="orderDialog = false">ยกเลิก</v-btn>
          <v-btn
            color="primary" class="text-none px-6" :loading="orderCreating"
            :disabled="!canCreateOrder" @click="createOrderFromChat"
          >สร้างออเดอร์</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Follow-up discount dialog -->
    <v-dialog v-model="discountDialog" max-width="420">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ตามดีลพร้อมส่วนลด</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-caption text-medium-emphasis mb-3">
            ระบบจะสร้างคูปองใช้ครั้งเดียวและใส่ส่วนลดให้ออเดอร์ #{{ discountTarget?.id }} ทันที
            — หน้าชำระเงินแสดงยอดใหม่เลย แล้วเตรียมข้อความตามดีลไว้ให้กดส่ง
          </div>
          <v-select v-model="discountChoice" :items="discountOptions" label="ส่วนลด" density="compact" class="mb-2" />
          <v-select
            v-model="discountHours"
            :items="[{ title: '24 ชั่วโมง', value: 24 }, { title: '48 ชั่วโมง', value: 48 }]"
            label="แจ้งลูกค้าว่าสิทธิ์ถึงภายใน" density="compact"
          />
          <v-alert v-if="discountError" type="error" variant="tonal" density="compact" class="mt-1">
            {{ discountError }}
          </v-alert>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="discountDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="discountSaving" @click="applyFollowupDiscount">
            ให้ส่วนลด
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="orderSnackbar" timeout="6000" color="success">
      {{ orderSnackbarText }}
      <template v-slot:actions>
        <v-btn variant="text" class="text-none" @click="copyPayURL">คัดลอกลิงก์</v-btn>
      </template>
    </v-snackbar>

    <!-- Link customer dialog -->
    <v-dialog v-model="linkDialog" max-width="440">
      <v-card>
        <v-card-title class="pa-5 pb-2">
          <span class="text-h6 font-weight-bold">ผูกแชทกับลูกค้า</span>
        </v-card-title>
        <v-card-text class="px-5">
          <div class="text-caption text-medium-emphasis mb-3">
            ผูกแล้วจะเห็นชื่อลูกค้าในแชท และใช้สร้างออเดอร์ให้ถูกคนได้เร็วขึ้น
          </div>
          <v-autocomplete
            v-model="linkCustomerId"
            :items="customers"
            item-title="name"
            item-value="id"
            label="เลือกลูกค้า"
            clearable
          >
            <template v-slot:item="{ item, props }">
              <v-list-item v-bind="props" :subtitle="item.raw.phone || '-'" />
            </template>
          </v-autocomplete>
        </v-card-text>
        <v-card-actions class="pa-5 pt-0">
          <v-spacer />
          <v-btn variant="text" class="text-none" @click="linkDialog = false">ยกเลิก</v-btn>
          <v-btn color="primary" class="text-none px-6" :loading="linking" @click="saveLink">บันทึก</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, computed, watch } from 'vue'
import { useDisplay } from 'vuetify'
import api from '@/services/api'

interface Conversation {
  id: number; platform: string; external_id: string;
  display_name: string; avatar_url: string;
  customer_id: number | null; customer?: { id: number; name: string } | null;
  unread_count: number; last_message_text: string; last_message_at: string;
  status: string; last_direction: string; waiting_since: string | null; tags: string[] | null;
  ai_disabled?: boolean;
}
interface CannedReply { id: number; title: string; text: string }
interface ChatMessage {
  id: number; conversation_id: number; direction: string;
  type: string; text: string; image_url: string; created_at: string;
  source?: string;
}

const { mobile } = useDisplay()

const conversations = ref<Conversation[]>([])
const activeConv = ref<Conversation | null>(null)
const messages = ref<ChatMessage[]>([])
const loading = ref(false)
const draft = ref('')
const sending = ref(false)
const sendError = ref('')
const threadEl = ref<HTMLElement | null>(null)

const webhookURL = computed(() => `${window.location.origin}/api/webhooks/…`)

// ── Filters: status tabs + platform + search ──
const statusTab = ref('waiting')
const platformFilter = ref('all')
const search = ref('')

const platformFilters = [
  { label: 'ทั้งหมด', value: 'all' },
  { label: 'LINE', value: 'line' },
  { label: 'Facebook', value: 'facebook' },
  { label: 'IG', value: 'instagram' },
]

function isWaiting(c: Conversation) {
  return c.status !== 'done' && c.last_direction === 'in'
}

const waitingCount = computed(() => conversations.value.filter(isWaiting).length)

const filteredConversations = computed(() => {
  let list = conversations.value
  if (statusTab.value === 'deals') list = followups.value.map(f => f.conversation)
  else if (statusTab.value === 'waiting') list = list.filter(isWaiting)
  else if (statusTab.value === 'active') list = list.filter(c => c.status !== 'done' && !isWaiting(c))
  else list = list.filter(c => c.status === 'done')
  if (platformFilter.value !== 'all') list = list.filter(c => c.platform === platformFilter.value)
  const q = (search.value || '').trim().toLowerCase()
  if (q) {
    list = list.filter(c =>
      (c.display_name || '').toLowerCase().includes(q) ||
      (c.last_message_text || '').toLowerCase().includes(q) ||
      (c.customer?.name || '').toLowerCase().includes(q))
  }
  // Waiting tab: longest-waiting first so the queue works oldest-up.
  if (statusTab.value === 'waiting') {
    return [...list].sort((a, b) =>
      new Date(a.waiting_since || a.last_message_at).getTime() -
      new Date(b.waiting_since || b.last_message_at).getTime())
  }
  return list
})

function waitingLabel(c: Conversation): string {
  if (!isWaiting(c) || !c.waiting_since) return ''
  const mins = Math.floor((Date.now() - new Date(c.waiting_since).getTime()) / 60000)
  if (mins < 1) return 'รอเมื่อครู่'
  if (mins < 60) return `รอ ${mins} นาที`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `รอ ${hours} ชม.`
  return `รอ ${Math.floor(hours / 24)} วัน`
}
function waitingUrgent(c: Conversation): boolean {
  if (!c.waiting_since) return false
  return Date.now() - new Date(c.waiting_since).getTime() > 60 * 60000
}

// ── "ตอบจากที่อื่นแล้ว" — LINE OA app replies never reach our webhook (LINE
// has no echo events, unlike Meta), so threads answered there stay "waiting"
// until manually cleared with this.
const answeredSaving = ref(false)
async function markAnswered() {
  if (!activeConv.value) return
  answeredSaving.value = true
  try {
    const { data } = await api.post(`/chats/${activeConv.value.id}/answered`)
    activeConv.value = data
    fetchConversations()
  } finally {
    answeredSaving.value = false
  }
}

// ── Status (จบงาน / เปิดใหม่) ──
const statusSaving = ref(false)
async function toggleStatus(status: 'open' | 'done') {
  if (!activeConv.value) return
  statusSaving.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/status`, { status })
    activeConv.value = data
    fetchConversations()
  } finally {
    statusSaving.value = false
  }
}

// ── Tags ──
const tagDialog = ref(false)
const tagDraft = ref<string[]>([])
const tagSaving = ref(false)
const tagSuggestions = ['รอโอน', 'โอนแล้ว', 'CF', 'รอของเข้า', 'ส่งแล้ว', 'ถามเฉยๆ']

function openTagDialog() {
  tagDraft.value = [...(activeConv.value?.tags || [])]
  tagDialog.value = true
}
async function saveTags() {
  if (!activeConv.value) return
  tagSaving.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/tags`, { tags: tagDraft.value })
    activeConv.value = data
    tagDialog.value = false
    fetchConversations()
  } finally {
    tagSaving.value = false
  }
}

// ── Canned replies ──
const cannedReplies = ref<CannedReply[]>([])
const cannedDialog = ref(false)
const cannedForm = ref<{ id: number | null; title: string; text: string } | null>(null)
const cannedSaving = ref(false)

async function fetchCanned() {
  const { data } = await api.get('/canned-replies')
  cannedReplies.value = data || []
}
function editCanned(cr: CannedReply | null) {
  cannedForm.value = cr ? { ...cr } : { id: null, title: '', text: '' }
}
async function saveCanned() {
  if (!cannedForm.value) return
  cannedSaving.value = true
  try {
    if (cannedForm.value.id) {
      await api.put(`/canned-replies/${cannedForm.value.id}`, cannedForm.value)
    } else {
      await api.post('/canned-replies', cannedForm.value)
    }
    cannedForm.value = null
    await fetchCanned()
  } finally {
    cannedSaving.value = false
  }
}
async function deleteCanned(cr: CannedReply) {
  await api.delete(`/canned-replies/${cr.id}`)
  await fetchCanned()
}

async function fetchConversations() {
  loading.value = true
  fetchFollowups()
  try {
    const { data } = await api.get('/chats')
    conversations.value = data || []
    // Keep the open thread's header info fresh (name/customer/unread).
    if (activeConv.value) {
      const updated = conversations.value.find(c => c.id === activeConv.value!.id)
      if (updated) activeConv.value = updated
    }
  } finally {
    loading.value = false
  }
}

async function openConversation(conv: Conversation) {
  activeConv.value = conv
  sendError.value = ''
  const { data } = await api.get(`/chats/${conv.id}/messages`)
  messages.value = data.messages || []
  scrollToBottom()
  if (conv.unread_count > 0) {
    conv.unread_count = 0
    api.post(`/chats/${conv.id}/read`)
  }
}

async function sendReply() {
  const text = draft.value.trim()
  if (!text || !activeConv.value) return
  sending.value = true
  sendError.value = ''
  try {
    const { data } = await api.post(`/chats/${activeConv.value.id}/reply`, { text })
    messages.value.push(data)
    draft.value = ''
    scrollToBottom()
    fetchConversations()
  } catch (err: any) {
    sendError.value = err.response?.data?.error || 'ส่งไม่สำเร็จ'
  } finally {
    sending.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (threadEl.value) threadEl.value.scrollTop = threadEl.value.scrollHeight
  })
}

// ── Link customer ──
const linkDialog = ref(false)
const linkCustomerId = ref<number | null>(null)
const linking = ref(false)
const customers = ref<any[]>([])

async function openLinkDialog() {
  if (!customers.value.length) {
    const { data } = await api.get('/customers')
    customers.value = data || []
  }
  linkCustomerId.value = activeConv.value?.customer_id || null
  linkDialog.value = true
}

async function saveLink() {
  if (!activeConv.value) return
  linking.value = true
  try {
    const { data } = await api.put(`/chats/${activeConv.value.id}/customer`, {
      customer_id: linkCustomerId.value || 0,
    })
    activeConv.value = data
    linkDialog.value = false
    fetchConversations()
  } finally {
    linking.value = false
  }
}

// ── Create order from chat (ปิดการขายในแชท) ──
interface OrderProduct {
  id: number; name: string; price: number; stock: number;
  variants?: { id: number; size: string; color: string; stock: number }[];
}
const orderDialog = ref(false)
const products = ref<OrderProduct[]>([])
const orderItems = ref<{ product_id: number; variant_id: number | null; quantity: number }[]>([])
const orderNotes = ref('')
const orderCoupon = ref('')
const appliedOrderCoupon = ref<{ code: string; discount: number } | null>(null)
const orderCouponError = ref('')
const orderCouponChecking = ref(false)
const orderCustomerIsMember = ref(false)
const orderSendLink = ref(true)
const orderCreating = ref(false)
const orderError = ref('')
const orderSnackbar = ref(false)
const orderSnackbarText = ref('')
const lastPayURL = ref('')

async function openOrderDialog() {
  orderItems.value = [{ product_id: 0, variant_id: null, quantity: 1 }]
  orderNotes.value = ''
  orderCoupon.value = ''
  appliedOrderCoupon.value = null
  orderCouponError.value = ''
  orderError.value = ''
  orderSendLink.value = true
  orderCustomerIsMember.value = false
  orderDialog.value = true
  if (!products.value.length) {
    const { data } = await api.get('/products')
    products.value = data || []
  }
  checkOrderMember()
}

// Member preview only — the backend recomputes the real discount (and
// auto-promotes returning customers) inside the order transaction.
async function checkOrderMember() {
  const customerId = activeConv.value?.customer_id
  if (!customerId) return
  if (!customers.value.length) {
    const { data } = await api.get('/customers')
    customers.value = data || []
  }
  const customer = customers.value.find(c => c.id === customerId)
  if (!customer) return
  if (customer.is_member) {
    orderCustomerIsMember.value = true
    return
  }
  if (customer.phone) {
    try {
      const { data } = await api.post('/shop/members/check', { phone: customer.phone })
      orderCustomerIsMember.value = !!data.is_member
    } catch { /* preview only */ }
  }
}

function variantsFor(productId: number) {
  return products.value.find(p => p.id === productId)?.variants || []
}
function variantLabel(v: { size: string; color: string; stock: number }) {
  const label = [v.size, v.color].filter(Boolean).join(' / ') || 'One size'
  return `${label} — เหลือ ${v.stock}`
}
function fmtMoney(n: number) {
  return n.toLocaleString('th-TH', { maximumFractionDigits: 2 })
}

const orderSubtotal = computed(() =>
  orderItems.value.reduce((sum, it) => {
    const p = products.value.find(pp => pp.id === it.product_id)
    return sum + (p ? p.price * (it.quantity || 0) : 0)
  }, 0))
const orderMemberDiscount = computed(() =>
  orderCustomerIsMember.value ? Math.round(orderSubtotal.value * 5) / 100 : 0)
const orderCouponDiscount = computed(() =>
  appliedOrderCoupon.value
    ? Math.min(appliedOrderCoupon.value.discount, orderSubtotal.value - orderMemberDiscount.value)
    : 0)
const orderTotal = computed(() =>
  Math.max(0, orderSubtotal.value - orderMemberDiscount.value - orderCouponDiscount.value))
const canCreateOrder = computed(() =>
  !!activeConv.value?.customer &&
  orderItems.value.length > 0 &&
  orderItems.value.every(it =>
    it.product_id && it.quantity > 0 &&
    (!variantsFor(it.product_id).length || it.variant_id != null)))

async function applyOrderCoupon() {
  const code = orderCoupon.value.trim().toUpperCase()
  if (!code || orderCouponChecking.value) return
  orderCouponChecking.value = true
  orderCouponError.value = ''
  try {
    const { data } = await api.post('/coupons/validate', {
      code, subtotal: orderSubtotal.value, customer_id: activeConv.value?.customer_id || 0,
    })
    appliedOrderCoupon.value = { code: data.code, discount: data.discount }
  } catch (err: any) {
    appliedOrderCoupon.value = null
    orderCouponError.value = err.response?.data?.error || 'ใช้โค้ดไม่สำเร็จ'
  } finally {
    orderCouponChecking.value = false
  }
}
function removeOrderCoupon() {
  appliedOrderCoupon.value = null
  orderCoupon.value = ''
  orderCouponError.value = ''
}

// ── Cross-sell suggestions in the order dialog (bought-together data) ──
const orderSuggestions = ref<{ id: number; name: string; price: number }[]>([])

watch(
  () => [orderDialog.value, orderItems.value.map(i => i.product_id).filter(Boolean).join(',')],
  async ([open, ids]) => {
    if (!open || !ids) {
      orderSuggestions.value = []
      return
    }
    try {
      const { data } = await api.get('/shop/products/suggest', { params: { ids } })
      const chosen = new Set(orderItems.value.map(i => i.product_id))
      orderSuggestions.value = (data || []).filter((p: any) => !chosen.has(p.id))
    } catch {
      orderSuggestions.value = []
    }
  },
)

function addSuggested(s: { id: number }) {
  // Reuse the first still-empty row if there is one, else append.
  const empty = orderItems.value.find(i => !i.product_id)
  if (empty) {
    empty.product_id = s.id
    empty.variant_id = null
  } else {
    orderItems.value.push({ product_id: s.id, variant_id: null, quantity: 1 })
  }
}

// The chat message pushed to the customer after the order is created —
// summary + payment link, composed from the server response so the amounts
// always match what the backend actually charged.
function composePaymentMessage(order: any, payURL: string): string {
  const lines: string[] = [`สรุปคำสั่งซื้อ Bruno Collective 🧾 (ออเดอร์ #${order.id})`]
  for (const it of order.items || []) {
    const variant = [it.size, it.color].filter(Boolean).join('/')
    lines.push(`• ${it.product?.name || 'สินค้า'}${variant ? ` (${variant})` : ''} x${it.quantity} — ฿${fmtMoney(it.price * it.quantity)}`)
  }
  if (order.member_discount || order.discount_amount) {
    lines.push(`ยอดรวม ฿${fmtMoney(order.subtotal || order.total_amount)}`)
    if (order.member_discount) lines.push(`ส่วนลดสมาชิก −฿${fmtMoney(order.member_discount)}`)
    if (order.discount_amount) lines.push(`คูปอง ${order.coupon_code} −฿${fmtMoney(order.discount_amount)}`)
  }
  lines.push(`ยอดชำระ ฿${fmtMoney(order.total_amount)}`)
  lines.push('')
  lines.push('ชำระเงินและแนบสลิปได้ที่ลิงก์นี้เลยค่ะ 👇')
  lines.push(payURL)
  return lines.join('\n')
}

async function createOrderFromChat() {
  if (!activeConv.value) return
  orderCreating.value = true
  orderError.value = ''
  try {
    const { data } = await api.post(`/chats/${activeConv.value.id}/order`, {
      notes: orderNotes.value.trim(),
      coupon_code: appliedOrderCoupon.value?.code || '',
      items: orderItems.value,
    })
    lastPayURL.value = data.pay_url
    const message = composePaymentMessage(data.order, data.pay_url)
    orderDialog.value = false
    orderSnackbarText.value = `สร้างออเดอร์ #${data.order.id} เรียบร้อย`
    orderSnackbar.value = true
    if (orderSendLink.value) {
      try {
        const { data: msg } = await api.post(`/chats/${activeConv.value.id}/reply`, { text: message })
        messages.value.push(msg)
        scrollToBottom()
        fetchConversations()
      } catch (err: any) {
        // Order exists but the chat push failed — stage the message in the
        // composer so the admin can retry with the send button.
        draft.value = message
        sendError.value = err.response?.data?.error || 'สร้างออเดอร์แล้ว แต่ส่งข้อความไม่สำเร็จ — กดส่งอีกครั้งได้เลย'
      }
    } else {
      draft.value = message
    }
  } catch (err: any) {
    orderError.value = err.response?.data?.error || 'สร้างออเดอร์ไม่สำเร็จ'
  } finally {
    orderCreating.value = false
  }
}

async function copyPayURL() {
  if (lastPayURL.value) await navigator.clipboard.writeText(lastPayURL.value)
}

// ── ดีลค้าง: unpaid chat orders + follow-up tools ──
interface DealOrder { id: number; total_amount: number; coupon_code: string; created_at: string; pay_url: string }
interface Followup { conversation: Conversation; orders: DealOrder[] }

const followups = ref<Followup[]>([])

async function fetchFollowups() {
  try {
    const { data } = await api.get('/chats/followups')
    followups.value = data || []
  } catch { /* non-critical — tab just shows empty */ }
}

const dealsCount = computed(() => followups.value.length)

const activeDeals = computed<DealOrder[]>(() => {
  if (!activeConv.value) return []
  return followups.value.find(f => f.conversation.id === activeConv.value!.id)?.orders || []
})

function dealLabel(conv: Conversation): string {
  const f = followups.value.find(x => x.conversation.id === conv.id)
  if (!f || !f.orders.length) return ''
  const o = f.orders[0]
  return `#${o.id} ฿${fmtMoney(o.total_amount)} · ${hoursSince(o.created_at)}`
}

function hoursSince(iso: string): string {
  const h = Math.floor((Date.now() - new Date(iso).getTime()) / 3600000)
  if (h < 1) return 'ไม่ถึง 1 ชม.'
  if (h < 24) return `${h} ชม.`
  return `${Math.floor(h / 24)} วัน ${h % 24} ชม.`
}

function draftFollowup(d: DealOrder) {
  draft.value = [
    `สวัสดีค่ะ ขอแจ้งเตือนออเดอร์ #${d.id} ยอด ฿${fmtMoney(d.total_amount)} ยังรอการชำระเงินอยู่ค่ะ 🙏`,
    'ชำระเงินและแนบสลิปได้ที่ลิงก์นี้เลยค่ะ 👇',
    d.pay_url,
    'ติดขัดตรงไหนทักถามได้เลยนะคะ',
  ].join('\n')
}

const discountDialog = ref(false)
const discountTarget = ref<DealOrder | null>(null)
const discountChoice = ref('percent:10')
const discountHours = ref(24)
const discountSaving = ref(false)
const discountError = ref('')
const discountOptions = [
  { title: 'ลด 5%', value: 'percent:5' },
  { title: 'ลด 10%', value: 'percent:10' },
  { title: 'ลด 15%', value: 'percent:15' },
  { title: 'ลด ฿50', value: 'fixed:50' },
  { title: 'ลด ฿100', value: 'fixed:100' },
]

function openDiscountDialog(d: DealOrder) {
  discountTarget.value = d
  discountChoice.value = 'percent:10'
  discountHours.value = 24
  discountError.value = ''
  discountDialog.value = true
}

async function applyFollowupDiscount() {
  if (!discountTarget.value) return
  const [type, valueStr] = discountChoice.value.split(':')
  discountSaving.value = true
  discountError.value = ''
  try {
    const { data } = await api.post(`/orders/${discountTarget.value.id}/followup-discount`, {
      type, value: Number(valueStr), expires_hours: discountHours.value,
    })
    const o = data.order
    const label = type === 'percent' ? `${valueStr}%` : `฿${valueStr}`
    const deadline = new Date(data.expires_at).toLocaleString('th-TH', {
      day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit',
    })
    draft.value = [
      `พิเศษสำหรับคุณลูกค้าค่ะ 💛 ออเดอร์ #${o.id} ลดเพิ่ม ${label}`,
      `เหลือชำระเพียง ฿${fmtMoney(o.total_amount)} (สิทธิ์นี้ถึง ${deadline} นะคะ)`,
      'ชำระเงินได้ที่ลิงก์เดิมเลยค่ะ 👇',
      data.pay_url,
    ].join('\n')
    discountDialog.value = false
    fetchFollowups()
  } catch (err: any) {
    discountError.value = err.response?.data?.error || 'ให้ส่วนลดไม่สำเร็จ'
  } finally {
    discountSaving.value = false
  }
}

// ── AI assistant (per-thread toggle) ──
const aiEnabled = ref(false)
const aiToggling = ref(false)

async function fetchAIStatus() {
  try {
    const { data } = await api.get('/chats/summary')
    aiEnabled.value = !!data.ai_enabled
  } catch { /* toggle just stays hidden */ }
}

async function toggleAI() {
  if (!activeConv.value) return
  aiToggling.value = true
  try {
    const { data } = await api.post(`/chats/${activeConv.value.id}/ai`)
    activeConv.value = data
    fetchConversations()
  } finally {
    aiToggling.value = false
  }
}

// Machine-sent outbound messages get a small label so the admin can tell who
// answered: keyword rule, AI assistant, or a broadcast campaign.
function sourceLabel(source?: string): string {
  return ({ rule: 'ตอบอัตโนมัติ', ai: 'AI', broadcast: 'Broadcast' } as Record<string, string>)[source || ''] || ''
}

// ── Realtime: WebSocket with polling fallback ──
const wsConnected = ref(false)
let ws: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let closed = false

function connectWS() {
  const token = localStorage.getItem('token')
  if (!token) return
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  ws = new WebSocket(`${proto}://${window.location.host}/api/ws/chat?token=${token}`)
  ws.onopen = () => { wsConnected.value = true }
  ws.onmessage = (ev) => {
    try {
      const payload = JSON.parse(ev.data)
      if (payload.type !== 'message') return
      if (activeConv.value && payload.conversation_id === activeConv.value.id) {
        // Skip our own replies — they're appended locally on send.
        if (!messages.value.some(m => m.id === payload.message.id)) {
          messages.value.push(payload.message)
          scrollToBottom()
        }
        if (payload.message.direction === 'in') {
          api.post(`/chats/${activeConv.value.id}/read`)
        }
      }
      fetchConversations()
    } catch {}
  }
  ws.onclose = () => {
    wsConnected.value = false
    if (!closed) reconnectTimer = setTimeout(connectWS, 5000)
  }
  ws.onerror = () => ws?.close()
}

// ── Helpers ──
function platformIcon(p: string) {
  return ({ line: 'mdi-chat', facebook: 'mdi-facebook-messenger', instagram: 'mdi-instagram' } as Record<string, string>)[p] || 'mdi-chat-outline'
}
function platformColor(p: string) {
  return ({ line: '#008300', facebook: '#2a78d6', instagram: '#4a3aa7' } as Record<string, string>)[p] || 'grey'
}
function timeAgo(iso: string) {
  const mins = Math.floor((Date.now() - new Date(iso).getTime()) / 60000)
  if (mins < 1) return 'เมื่อครู่'
  if (mins < 60) return `${mins} นาที`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} ชม.`
  return `${Math.floor(hours / 24)} วัน`
}
function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  fetchConversations()
  fetchCanned()
  fetchAIStatus()
  connectWS()
  pollTimer = setInterval(() => {
    if (!wsConnected.value) fetchConversations()
  }, 20_000)
})

onUnmounted(() => {
  closed = true
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (pollTimer) clearInterval(pollTimer)
  ws?.close()
})
</script>

<style scoped>
.chat-shell {
  display: flex;
  height: calc(100vh - 210px);
  min-height: 420px;
}
.conv-pane {
  width: 320px;
  min-width: 320px;
  border-right: 1px solid #E5E7EB;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.conv-list {
  flex: 1;
  overflow-y: auto;
}
.thread-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.conv-item {
  display: flex;
  align-items: center;
  cursor: pointer;
  border-bottom: 1px solid #F1F3F5;
  transition: background 0.15s;
}
.conv-item:hover {
  background: #F6F7F9;
}
.conv-item--active {
  background: rgba(196, 162, 77, 0.10);
}
.conv-item-body {
  flex: 1;
  min-width: 0;
}
.thread-messages {
  flex: 1;
  overflow-y: auto;
  background: #F6F7F9;
}
.bubble {
  max-width: 70%;
  padding: 8px 12px;
  border-radius: 14px;
}
.bubble--in {
  background: #FFFFFF;
  border: 1px solid #E5E7EB;
  border-top-left-radius: 4px;
}
.bubble--out {
  background: #1A1714;
  color: #fff;
  border-top-right-radius: 4px;
}
.bubble--out .bubble-time {
  color: rgba(255, 255, 255, 0.55);
}
.bubble-time {
  font-size: 10px !important;
  color: #9CA3AF;
  text-align: right;
  margin-top: 2px;
}
@media (max-width: 960px) {
  .conv-pane {
    width: 100%;
    min-width: 0;
    border-right: none;
  }
  .conv-pane--hidden {
    display: none;
  }
  .thread-pane--hidden {
    display: none;
  }
}
</style>
