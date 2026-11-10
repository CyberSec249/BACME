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
          <SolutionOutlined />
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
          <a-typography-title style="text-align: center; margin-bottom: 20px">Emergency Certificate Operations</a-typography-title>
        </a-typography>
        <a-alert message="This page is for emergency certificate operations after the management account is lost" type="info" closable show-icon style="margin-bottom: 10px"/>
        <a-card title="Files to Revoke">
          <a-row>
            <a-col :span="6" style="text-align: center">
            <!-- Upload area for file 1 -->
              <a-upload
                    v-model:fileList="fileList1"
                    name="file1"
                    :max-count="1"
                    :before-upload="() => false"
                @change="handleChange1"
              >
                <a-button>
                  <upload-outlined /> Select the certificate file to revoke
                </a-button>
              </a-upload>
            </a-col>
            <a-col :span="6" style="text-align: center">
              <!-- Upload area for file 2 -->
              <a-upload
                  v-model:fileList="fileList2"
                  name="file2"
                  :max-count="1"
                  :before-upload="() => false"
                @change="handleChange2"
              >
                <a-button>
                  <upload-outlined /> Select the private key file of the certificate
                </a-button>
              </a-upload>
            </a-col>
            <a-col :span="6" style="text-align: center">
              <!-- Start buttons -->
              <a-button
                  :disabled='(!fileList1.length || !fileList2.length)'
                  @click="handleStart1"
              >
                Revoke with Private Key Proof
              </a-button>
            </a-col>
            <a-col :span="6" style="text-align: center">
              <!-- Start buttons -->
              <a-button
                  :disabled='!fileList1.length || fileList2.length'
                  @click="handleStart2"
              >
                Revoke with Domain Control Validation
              </a-button>
            </a-col>
          </a-row>
        </a-card>
        <a-card title="Emergency Operation Progress" :bordered="false" style="margin-top: 15px">
          <a-progress :percent="percent" :status="status" />
        </a-card>
        <a-card title="Emergency Revocation History" style="margin-top: 15px">
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
  UploadOutlined,
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
import {onMounted, h, ref} from 'vue';
import axios from 'axios';
import {message, notification} from 'ant-design-vue';
import updateLog from "@/components/updateLog.vue";
const selectedKeys = ref(['5']);
const collapsed = ref(false);

// Drawer
const open = ref(false);
const afterOpenChange = bool => {
  console.log('open', bool);
};
const showDrawer = () => {
  open.value = true;
};

// File lists
const fileList1 = ref([]); // file 1
const fileList2 = ref([]); // file 2

// Handle changes to the file 1 selection
const handleChange1 = ({ fileList }) => {
  fileList1.value = fileList;
};

// Handle changes to the file 2 selection
const handleChange2 = ({ fileList }) => {
  fileList2.value = fileList;
};

// Handle clicks on the start buttons
const handleStart1 = async () => {
  try {
    // Create the FormData object
    const formData = new FormData();
    formData.append('file1',  fileList1.value[0].originFileObj);
    formData.append('file2',  fileList2.value[0].originFileObj);
    // console.log( fileList1.value[0].originFileObj);
    console.log(formData.get('file1'));
    console.log(formData.get('file2'));
    startProgress()
    // Send the certificate revocation request to the backend
    const response = await axios.post('http://localhost:8080/api/revokeCertificateEmergency?type=0', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    // Show the success message
    percent.value = 100;
    status.value = "";
    isRunning.value = false;
    openNotification("Certificate Revocation", response.data.message);
    await fetchCertInfo()
  } catch (error) {
    // Show the error message
    clearInterval(timer);
    status.value = "exception"; // progress bar shows failure
    isRunning.value = false;
    message.error(error.message);
  }
};

// Handle clicks on the start buttons
const handleStart2 = async () => {
  try {
    // Create the FormData object
    const formData = new FormData();
    formData.append('file1',  fileList1.value[0].originFileObj);
    startProgress()
    // Send the certificate revocation request to the backend
    const response = await axios.post('http://localhost:8080/api/revokeCertificateEmergency?type=1', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    // Show the success message
    percent.value = 100;
    status.value = "";
    isRunning.value = false;
    openNotification("Certificate Revocation", response.data.message);
    await fetchCertInfo()
  } catch (error) {
    // Show the error message
    clearInterval(timer);
    status.value = "exception"; // progress bar shows failure
    isRunning.value = false;
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

// Progress bar
const percent = ref(0); // progress percentage
const isRunning = ref(false); // whether it is running
const status = ref("active"); // animated
let timer = null; // timer
const startProgress = () => {
  if (isRunning.value) return; // return if already running
  isRunning.value = true;
  percent.value = 0; // reset to 0

  timer = setInterval(() => {
    if (percent.value < 90) {
      percent.value += 10; // increase by 10% each time
    }
    else if (percent.value >= 90 && percent.value < 99) {
      percent.value += 3;
    }
    else {
      clearInterval(timer); // stop at 99% and wait for the backend
    }
  }, 300); // update every 300 ms
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
    title: 'Revocation Time',
    dataIndex: 'revokeTime',
    key: 'revokeTime',
  },
];

// Fetch the table data
const fetchCertInfo = async () => {
  try {
    const response = await axios.get('http://localhost:8080/api/getRevokedCertInfo');
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