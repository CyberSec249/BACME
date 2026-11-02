<!--Author: A1andNS-->
<template>
  <a-layout>
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible>
      <div class="logo">
        <img src="../assets/logo.png" alt="logo走丢了" style="width: 100%;height:100%" />
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
        <a-menu-item key="1">
          <HomeOutlined />
          <span><router-link to="/">概览</router-link></span>
        </a-menu-item>
        <a-menu-item key="2">
          <FileProtectOutlined  />
          <span><router-link to="/controlValidation">控制权验证</router-link></span>
        </a-menu-item>
        <a-menu-item key="3">
          <FileAddOutlined />
          <span><router-link to="/apply">证书申请</router-link></span>
        </a-menu-item>
        <a-menu-item key="4">
          <AuditOutlined />
          <span><router-link to="/management">证书维护</router-link></span>
        </a-menu-item>
        <a-menu-item key="5">
          <SolutionOutlined/>
          <span><router-link to="/emergencyOperations">应急操作</router-link></span>
        </a-menu-item>
        <a-menu-item key="6">
          <CheckCircleOutlined />
          <span><router-link to="/verify">证书验证</router-link></span>
        </a-menu-item>
        <a-menu-item key="7">
          <InfoCircleOutlined />
          <span><router-link to="/about">关于系统</router-link></span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <!-- 右边内容区域 -->
    <a-layout>
      <a-layout-header style="background: #fff; padding: 0">
        <menu-unfold-outlined
            v-if="collapsed"
            class="trigger"
            @click="() => (collapsed = !collapsed)"
        />
        <menu-fold-outlined v-else class="trigger" @click="() => (collapsed = !collapsed)" />
      </a-layout-header>
      <a-layout-content
          :style="{ margin: '24px 16px', padding: '24px', minHeight: '640px' }"
      >
        <a-typography>
          <a-typography-title style="text-align: center; margin-bottom: 20px">证书验证</a-typography-title>
        </a-typography>
        <a-alert message="该页面主要用于演示证书验证" type="info" closable show-icon style="margin-bottom: 10px"/>
        <a-row>
          <a-col :span="8">
            <a-card title="待验证证书选择区" style="margin-bottom: 10px; margin-right: 5px; min-height: 550px">
              <a-upload-dragger
                  v-model:fileList="fileList"
                  name="file"
                  @change="handleChange"
                  :customRequest="handleStart"
                  :max-count="1"
              >
                <p class="ant-upload-drag-icon" style="margin-top: 125px">
                  <inbox-outlined></inbox-outlined>
                </p>
                <p class="ant-upload-text">点击或拖拽证书到此处验证</p>
                <p class="ant-upload-hint" style="margin-bottom: 125px">
                  请选择一个需要验证的证书到此处
                </p>
              </a-upload-dragger>
            </a-card>
          </a-col>
          <a-col :span="16">
            <a-card title="证书解析内容" style="margin-bottom: 10px; margin-left: 5px; min-height: 550px">
              <p>版本：{{ result.certInfo.version }}</p>
              <p>序列号：{{ result.certInfo.serial }}</p>
              <p>签名算法：{{ result.certInfo.signAlg }}</p>
              <p>颁发者：{{ result.certInfo.issuer }}</p>
              <p>起始有效时间：{{ result.certInfo.notBefore }}</p>
              <p>终止有效时间：{{ result.certInfo.notAfter }}</p>
              <p>证书主体：{{ result.certInfo.subject }}</p>
              <p>证书公钥：{{ result.certInfo.pk }}</p>
              <p>证书主体替换名称：{{ result.certInfo.san }}</p>
              <p>证书类型：{{result.certInfo.certType}}</p>
              <p>证书验证结果：{{result.certInfo.status}}</p>
            </a-card>
          </a-col>
        </a-row>
        <a-card title="证书验证历史">
          <a-table :columns="columns" :data-source="data" :bordered="true" size="small">
          </a-table>
        </a-card>
      </a-layout-content>
      <!--   内容底部   -->
      <a-layout-footer style="text-align: center">
        DPKI-ACME原型系统 ©2024 Created by A1andNS 西安电子科技大学
      </a-layout-footer>
    </a-layout>
  </a-layout>
  <!--  抽屉-->
  <a-drawer
      v-model:open="open"
      class="custom-class"
      root-class-name="root-class-name"
      :root-style="{ color: 'blue' }"
      style="color: Black"
      title="更新日志"
      placement="right"
      @after-open-change="afterOpenChange"
  >
    <a-row :gutter="[16,16]">
      <a-col :span="24" v-for="log in updateLog.data().updateLogs">
        <a-card v-bind:title=log.title :bordered="true">
          <p>{{log.content}}</p>
        </a-card>
      </a-col>
    </a-row>
  </a-drawer>
  <!--  浮动按钮-->
  <a-float-button tooltip="更新日志" :style="{right: '24px',}" @click="showDrawer"></a-float-button>
</template>

<script setup>
import {SolutionOutlined,
  InboxOutlined,
  SmileOutlined,
  FileProtectOutlined,
  FileAddOutlined,
  HomeOutlined,
  AuditOutlined,
  InfoCircleOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  CheckCircleOutlined
} from '@ant-design/icons-vue';
import {onMounted, h, ref, reactive} from 'vue';
import axios from 'axios';
import {message, notification} from 'ant-design-vue';
import updateLog from "@/components/updateLog.vue";
const selectedKeys = ref(['6']);
const collapsed = ref(false);

// 抽屉
const open = ref(false);
const afterOpenChange = bool => {
  console.log('open', bool);
};
const showDrawer = () => {
  open.value = true;
};

// 显示证书解析内容
// 对象类型
const result = reactive({
  certInfo: {
    "version": "待解析",
    "serial": '待解析',
    "signAlg": '待解析',
    "subject": '待解析',
    "issuer": '待解析',
    "notBefore": '待解析',
    "notAfter": '待解析',
    "pk": '待解析',
    "san": '待解析',
    "certType": '待解析',
    "status":"待检查"
  },
});
// 文件列表
const fileList = ref([]);

// 处理文件1选择变化
const handleChange = ({ fileList }) => {
  fileList.value = fileList;
};

// 处理开始按钮点击
const handleStart = async ({ file, onSuccess, onError }) => {
  try {
    // 创建 FormData 对象
    const formData = new FormData();
    formData.append('file', file);
    console.log(formData.get('file'));
    message.info("发起证书验证请求")
    // 发送证书撤销请求到后端
    const response = await axios.post('http://localhost:8080/api/verify', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    // 显示成功消息
    openNotification("证书验证通知", "验证结果已获得");
    result.certInfo = response.data;
    onSuccess('证书解析成功', file);
    await fetchCertInfo()
  } catch (error) {
    // 显示错误消息
    onError(error);
    message.error(error.message);
  }
};

// 通知
const openNotification = (title,content) => {
  notification.open({
    message: title,
    description:
    content,
    icon: () =>
        h(SmileOutlined, {
          style: 'color: #108ee9',
        })
  });
};

// 对象类型
const data = ref([]);

const columns = [
  {
    title: '证书Hash值',
    dataIndex: 'certHash',
    key: 'certHash',
  },
  {
    title: '标识符',
    dataIndex: 'IDs',
    key: 'IDs',
    ellipsis: true
  },
  {
    title: '验证时间',
    dataIndex: 'verifiedTime',
    key: 'verifiedTime',
  },
  {
    title: "验证结果",
    dataIndex: 'verifiedResult',
    key: 'verifiedResult',
  }
];

// 获取table数据
const fetchCertInfo = async () => {
  try {
    const response = await axios.get('http://localhost:8080/api/getVerifiedCertInfo');
    data.value = response.data;
  } catch (error) {
    message.error('无法获取证书信息:', error)
  }
};
onMounted(()=>{
  fetchCertInfo();
});
</script>

<style>
.trigger {
  font-size: 18px;
  line-height: 64px;
  padding: 0 24px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.logo {
  height: 32px;
  background: rgba(255, 255, 255, 0.3);
  margin: 16px;
}

.site-layout {
  background: #fff;
}
</style>