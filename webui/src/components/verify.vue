<!--Author: A1andNS-->
<template>
  <a-layout>
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible>
      <div class="logo">
        <img src="../assets/logo.png" alt="logo not found" style="width: 100%;height:100%" />
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
        <a-menu-item key="1">
          <HomeOutlined />
          <span><router-link to="/">Overview</router-link></span>
        </a-menu-item>
        <a-menu-item key="2">
          <FileProtectOutlined  />
          <span><router-link to="/controlValidation">Control Validation</router-link></span>
        </a-menu-item>
        <a-menu-item key="3">
          <FileAddOutlined />
          <span><router-link to="/apply">Certificate Application</router-link></span>
        </a-menu-item>
        <a-menu-item key="4">
          <AuditOutlined />
          <span><router-link to="/management">Certificate Management</router-link></span>
        </a-menu-item>
        <a-menu-item key="5">
          <SolutionOutlined/>
          <span><router-link to="/emergencyOperations">Emergency Operations</router-link></span>
        </a-menu-item>
        <a-menu-item key="6">
          <CheckCircleOutlined />
          <span><router-link to="/verify">Certificate Verification</router-link></span>
        </a-menu-item>
        <a-menu-item key="7">
          <InfoCircleOutlined />
          <span><router-link to="/about">About</router-link></span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <!-- Content area on the right -->
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
          <a-typography-title style="text-align: center; margin-bottom: 20px">Certificate Verification</a-typography-title>
        </a-typography>
        <a-alert message="This page demonstrates certificate verification" type="info" closable show-icon style="margin-bottom: 10px"/>
        <a-row>
          <a-col :span="8">
            <a-card title="Certificate to Verify" style="margin-bottom: 10px; margin-right: 5px; min-height: 550px">
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
                <p class="ant-upload-text">Click or drag a certificate here to verify it</p>
                <p class="ant-upload-hint" style="margin-bottom: 125px">
                  Select one certificate to verify
                </p>
              </a-upload-dragger>
            </a-card>
          </a-col>
          <a-col :span="16">
            <a-card title="Parsed Certificate" style="margin-bottom: 10px; margin-left: 5px; min-height: 550px">
              <p>Version: {{ result.certInfo.version }}</p>
              <p>Serial number: {{ result.certInfo.serial }}</p>
              <p>Signature algorithm: {{ result.certInfo.signAlg }}</p>
              <p>Issuer: {{ result.certInfo.issuer }}</p>
              <p>Not before: {{ result.certInfo.notBefore }}</p>
              <p>Not after: {{ result.certInfo.notAfter }}</p>
              <p>Subject: {{ result.certInfo.subject }}</p>
              <p>Public key: {{ result.certInfo.pk }}</p>
              <p>Subject alternative names: {{ result.certInfo.san }}</p>
              <p>Certificate type: {{result.certInfo.certType}}</p>
              <p>Verification result: {{result.certInfo.status}}</p>
            </a-card>
          </a-col>
        </a-row>
        <a-card title="Verification History">
          <a-table :columns="columns" :data-source="data" :bordered="true" size="small">
          </a-table>
        </a-card>
      </a-layout-content>
      <!--   Footer   -->
      <a-layout-footer style="text-align: center">
        DPKI-ACME Prototype ©2024 Created by A1andNS, Xidian University
      </a-layout-footer>
    </a-layout>
  </a-layout>
  <!--  Drawer-->
  <a-drawer
      v-model:open="open"
      class="custom-class"
      root-class-name="root-class-name"
      :root-style="{ color: 'blue' }"
      style="color: Black"
      title="Changelog"
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
  <!--  Floating button-->
  <a-float-button tooltip="Changelog" :style="{right: '24px',}" @click="showDrawer"></a-float-button>
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

// Drawer
const open = ref(false);
const afterOpenChange = bool => {
  console.log('open', bool);
};
const showDrawer = () => {
  open.value = true;
};

// Show the parsed certificate
// Object type
const result = reactive({
  certInfo: {
    "version": "Pending",
    "serial": 'Pending',
    "signAlg": 'Pending',
    "subject": 'Pending',
    "issuer": 'Pending',
    "notBefore": 'Pending',
    "notAfter": 'Pending',
    "pk": 'Pending',
    "san": 'Pending',
    "certType": 'Pending',
    "status":"Pending"
  },
});
// File lists
const fileList = ref([]);

// Handle changes to the file 1 selection
const handleChange = ({ fileList }) => {
  fileList.value = fileList;
};

// Handle clicks on the start buttons
const handleStart = async ({ file, onSuccess, onError }) => {
  try {
    // Create the FormData object
    const formData = new FormData();
    formData.append('file', file);
    console.log(formData.get('file'));
    message.info("Certificate verification requested")
    // Send the certificate revocation request to the backend
    const response = await axios.post('http://localhost:8080/api/verify', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    // Show the success message
    openNotification("Certificate Verification", "Verification result received");
    result.certInfo = response.data;
    onSuccess('Certificate parsed successfully', file);
    await fetchCertInfo()
  } catch (error) {
    // Show the error message
    onError(error);
    message.error(error.message);
  }
};

// Notification
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

// Object type
const data = ref([]);

const columns = [
  {
    title: 'Certificate Hash',
    dataIndex: 'certHash',
    key: 'certHash',
  },
  {
    title: 'Identifiers',
    dataIndex: 'IDs',
    key: 'IDs',
    ellipsis: true
  },
  {
    title: 'Verification Time',
    dataIndex: 'verifiedTime',
    key: 'verifiedTime',
  },
  {
    title: "Verification Result",
    dataIndex: 'verifiedResult',
    key: 'verifiedResult',
  }
];

// Fetch the table data
const fetchCertInfo = async () => {
  try {
    const response = await axios.get('http://localhost:8080/api/getVerifiedCertInfo');
    data.value = response.data;
  } catch (error) {
    message.error('Failed to fetch certificate information:', error)
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