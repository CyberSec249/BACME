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
          <a-typography-title style="text-align: center; margin-bottom: 20px">Control Validation</a-typography-title>
        </a-typography>
        <a-card title="Control Validation Form" :bordered="false">
          <a-form :model="form" @submit="handleSubmit">
            <a-form-item label="ID" :label-col="{ span: 4 }" :wrapper-col="{ span: 16 }">
              <a-input v-model:value="form.ids" placeholder="Enter all IDs to validate" />
              <p>Note: enter one or more IDs separated by commas, e.g., www.example1.com or www.example1.com,www.example2.com,127.0.0.1</p>
            </a-form-item>
            <a-form-item :wrapper-col="{ offset: 4, span: 16 }">
              <a-button type="primary" html-type="submit">Submit</a-button>
            </a-form-item>
          </a-form>
        </a-card>
        <a-card title="Control Validation Progress" :bordered="false" style="margin-top: 15px">
          <a-progress :percent="percent" :status="status" />
        </a-card>
        <a-card title="Obtained Control Rights" style="margin-top: 15px">
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
  const selectedKeys = ref(['2']);
  const collapsed = ref(false);

  // Drawer
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
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

  // Form data
  const form = ref({
    ids: '',
  });

  // Submit the form
  const handleSubmit = async (e) => {
    e.preventDefault(); // prevent the default form submission
    try {
      // Send a POST request to the backend
      message.info("Request submitted")
      startProgress()
      const response = await axios.post('http://localhost:8080/api/idControlRight', form.value);
      // Show the success message
      percent.value = 100;
      status.value = "";
      isRunning.value = false;
      openNotification("Control Validation",response.data.message);
      await fetchIDControlRight();
    } catch (error) {
      // Show the error message
      clearInterval(timer);
      status.value = "exception"; // progress bar shows failure
      isRunning.value = false;
      message.error('Submission failed: ' + error.message);
    }
  };

  // Object type
  const data = ref([]);

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: 'Valid Until',
      dataIndex: 'validTime',
      key: 'validTime',
    },
  ];
  // Fetch the table data
  const fetchIDControlRight = async () => {
    try {
      const response = await axios.get('http://localhost:8080/api/idControlRightInfo');
      data.value = response.data;
    } catch (error) {
      message.error('Failed to fetch control right information:', error)
    }
  };
  onMounted(()=>{
    fetchIDControlRight();
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