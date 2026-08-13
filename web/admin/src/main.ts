import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import './styles/admin-theme.css'
import router from './router'
import App from './App.vue'
import { loadMediaBaseURL } from './utils/mediaUrl'

async function bootstrap() {
  await loadMediaBaseURL()

  const app = createApp(App)
  const pinia = createPinia()

  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
  }

  app.use(pinia)
  app.use(router)
  app.use(ElementPlus, { size: 'default' })
  app.mount('#app')
}

void bootstrap()
