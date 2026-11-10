// Author: A1andNS
import { createMemoryHistory, createRouter } from 'vue-router';
import { createApp } from 'vue';
import IndexView from './components/index.vue';
import AboutView from './components/about.vue';
import ApplyView from './components/apply.vue';
import ManagementView from './components/management.vue';
import antd from 'ant-design-vue';
import 'ant-design-vue/dist/reset.css';
import App from "@/app.vue";
import dayjs from 'dayjs';
import ControlValidtionView from "@/components/ControlValidtionView.vue";
import EmergencyOperations from "@/components/emergencyOperations.vue";
import Verify from "@/components/verify.vue";

dayjs.locale('en');

const routes = [
    { path: '/', component: IndexView },
    { path: '/controlValidation', component: ControlValidtionView },
    { path: '/apply', component: ApplyView },
    { path: '/management', component: ManagementView },
    { path: '/about', component: AboutView },
    { path: '/emergencyOperations', component: EmergencyOperations },
    { path: '/verify', component: Verify },
]

const router = createRouter({
    history: createMemoryHistory(),
    routes,
})

const app = createApp(App)
app.use(router)
app.use(antd)
app.mount('#app')
