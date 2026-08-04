<template>
  <div class="login-container">
    <span class="ambient ambient-one" />
    <span class="ambient ambient-two" />
    <div class="login-shell">
      <section class="brand-panel">
        <div class="brand-logo">
          <span><el-icon><VideoCameraFilled /></el-icon></span>
          <div><strong>AI VIDEO</strong><small>ADMIN CENTER</small></div>
        </div>
        <div class="brand-copy">
          <div class="brand-kicker"><i /> INTELLIGENT OPERATIONS</div>
          <h1>让内容运营<br><em>更清晰、更高效</em></h1>
          <p>集中管理用户、订阅、视频任务与业务配置，掌握平台每一步运行状态。</p>
        </div>
        <div class="brand-features">
          <span><el-icon><DataAnalysis /></el-icon>经营数据概览</span>
          <span><el-icon><VideoPlay /></el-icon>生成任务管理</span>
          <span><el-icon><Lock /></el-icon>安全权限控制</span>
        </div>
        <span class="brand-orbit orbit-one" />
        <span class="brand-orbit orbit-two" />
      </section>

      <section class="login-panel">
        <div class="mobile-brand"><span><el-icon><VideoCameraFilled /></el-icon></span> AI VIDEO</div>
        <div class="login-heading">
          <span>ADMIN SIGN IN</span>
          <h2>欢迎回来</h2>
          <p>请输入管理员账号，继续进入管理后台</p>
        </div>
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @keyup.enter="handleLogin">
          <el-form-item label="管理员账号" prop="username">
            <el-input v-model="form.username" placeholder="请输入用户名" :prefix-icon="User" size="large" />
          </el-form-item>
          <el-form-item label="登录密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              size="large"
              show-password
            />
          </el-form-item>
          <el-form-item class="submit-item">
            <el-button class="login-button" type="primary" size="large" :loading="loading" @click="handleLogin">
              登录管理后台<el-icon><Right /></el-icon>
            </el-button>
          </el-form-item>
        </el-form>
        <div class="login-footer"><span />仅限授权人员访问<span /></div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function handleLogin() {
  await formRef.value?.validate()
  loading.value = true
  try {
    await userStore.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/')
  } catch {
    // error handled in interceptor
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  position: relative;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  overflow: hidden;
  background:
    linear-gradient(rgba(28, 53, 88, .035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(28, 53, 88, .035) 1px, transparent 1px),
    #f1f5fa;
  background-size: 36px 36px;
}
.ambient { position: absolute; border-radius: 50%; pointer-events: none; filter: blur(1px); }
.ambient-one { top: -220px; right: -130px; width: 480px; height: 480px; background: radial-gradient(circle, rgba(62, 143, 237, .14), transparent 68%); }
.ambient-two { bottom: -240px; left: -160px; width: 520px; height: 520px; background: radial-gradient(circle, rgba(54, 184, 203, .1), transparent 68%); }
.login-shell {
  position: relative;
  z-index: 1;
  display: grid;
  width: min(960px, 100%);
  min-height: 580px;
  grid-template-columns: 1.05fr .95fr;
  overflow: hidden;
  border: 1px solid rgba(220, 228, 238, .9);
  border-radius: 22px;
  background: #fff;
  box-shadow: 0 28px 80px rgba(25, 47, 78, .16);
}
.brand-panel {
  position: relative;
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: 42px 46px;
  overflow: hidden;
  background:
    radial-gradient(circle at 85% 12%, rgba(72, 180, 242, .19), transparent 28%),
    linear-gradient(142deg, #101d35 0%, #142a49 58%, #194363 100%);
  color: #fff;
}
.brand-panel::after {
  position: absolute;
  right: -120px;
  bottom: -150px;
  width: 370px;
  height: 370px;
  border: 1px solid rgba(134, 214, 255, .15);
  border-radius: 50%;
  content: '';
}
.brand-logo { position: relative; z-index: 2; display: flex; align-items: center; gap: 11px; }
.brand-logo > span, .mobile-brand > span {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid rgba(142, 214, 255, .3);
  border-radius: 11px;
  background: linear-gradient(145deg, #3388f1, #53b9e8);
  box-shadow: 0 8px 20px rgba(26, 119, 211, .3);
  font-size: 18px;
}
.brand-logo div { display: flex; flex-direction: column; gap: 3px; }
.brand-logo strong { font-size: 14px; letter-spacing: 1.6px; }
.brand-logo small { color: #839ab8; font-size: 8px; font-weight: 650; letter-spacing: 2px; }
.brand-copy { position: relative; z-index: 2; margin: auto 0; }
.brand-kicker { display: flex; align-items: center; gap: 8px; color: #91abc9; font-size: 9px; font-weight: 700; letter-spacing: 1.7px; }
.brand-kicker i { width: 7px; height: 7px; border-radius: 50%; background: #4bd3a7; box-shadow: 0 0 0 5px rgba(75, 211, 167, .11); }
.brand-copy h1 { margin: 18px 0 16px; font-size: 35px; font-weight: 720; line-height: 1.35; letter-spacing: -1px; }
.brand-copy h1 em { color: #83d6fa; font-style: normal; }
.brand-copy p { max-width: 365px; margin: 0; color: #adbed2; font-size: 13px; line-height: 1.85; }
.brand-features { position: relative; z-index: 2; display: flex; flex-wrap: wrap; gap: 18px; color: #98adc6; font-size: 10px; }
.brand-features span { display: flex; align-items: center; gap: 6px; }
.brand-features .el-icon { color: #70c8ef; font-size: 13px; }
.brand-orbit { position: absolute; border: 1px solid rgba(135, 211, 250, .1); border-radius: 50%; }
.orbit-one { top: 80px; right: -110px; width: 280px; height: 280px; }
.orbit-two { right: -20px; bottom: 60px; width: 115px; height: 115px; }
.login-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 60px 58px;
  background: #fff;
}
.mobile-brand { display: none; align-items: center; gap: 10px; color: #253d60; font-size: 14px; font-weight: 750; letter-spacing: 1.4px; }
.login-heading { margin-bottom: 30px; }
.login-heading > span { color: #4489e8; font-size: 9px; font-weight: 750; letter-spacing: 1.6px; }
.login-heading h2 { margin: 9px 0 7px; color: #202d41; font-size: 28px; font-weight: 730; letter-spacing: -.7px; }
.login-heading p { margin: 0; color: #96a1b1; font-size: 12px; }
.login-panel :deep(.el-form-item) { margin-bottom: 21px; }
.login-panel :deep(.el-form-item__label) { height: auto; padding-bottom: 7px; color: #596579; font-size: 12px; font-weight: 600; line-height: 1.4; }
.login-panel :deep(.el-input__wrapper) { min-height: 46px; padding: 1px 13px; border-radius: 9px; background: #f9fbfd; box-shadow: 0 0 0 1px #dfe6ee inset; }
.login-panel :deep(.el-input__wrapper:hover) { box-shadow: 0 0 0 1px #b8c9dc inset; }
.login-panel :deep(.el-input__wrapper.is-focus) { background: #fff; box-shadow: 0 0 0 1px #3479ed inset, 0 0 0 3px rgba(52, 121, 237, .09); }
.submit-item { margin-top: 6px; }
.login-button { width: 100%; height: 46px; margin-top: 2px; border-radius: 9px; letter-spacing: .3px; }
.login-button .el-icon { margin-left: 4px; }
.login-footer { display: flex; align-items: center; gap: 10px; margin-top: 7px; color: #a8b1be; font-size: 9px; text-align: center; }
.login-footer span { flex: 1; height: 1px; background: #edf1f5; }

@media (max-width: 800px) {
  .login-container { padding: 20px; }
  .login-shell { max-width: 460px; min-height: auto; grid-template-columns: 1fr; }
  .brand-panel { display: none; }
  .login-panel { padding: 45px 42px; }
  .mobile-brand { display: flex; margin-bottom: 44px; }
}

@media (max-width: 480px) {
  .login-container { padding: 12px; }
  .login-shell { border-radius: 17px; }
  .login-panel { padding: 34px 25px; }
  .mobile-brand { margin-bottom: 36px; }
  .login-heading h2 { font-size: 25px; }
}
</style>
