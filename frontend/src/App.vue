<script setup lang="ts">
import { NConfigProvider, darkTheme, NMessageProvider } from 'naive-ui'
import AppContent from './AppContent.vue'
</script>

<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <AppContent />
    </n-message-provider>
  </n-config-provider>
</template>

<script lang="ts">
import type { GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#D0BCFF',
    primaryColorHover: '#E8DEF8',
    primaryColorPressed: '#B69DF8',
    primaryColorSuppl: '#D0BCFF',
    bodyColor: '#1C1B1F',
    cardColor: '#2B2930',
    modalColor: '#2B2930',
    popoverColor: '#2B2930',
    tableColor: '#2B2930',
    inputColor: '#1C1B1F',
    actionColor: '#1C1B1F',
    borderColor: '#454349',
    dividerColor: '#454349',
    hoverColor: 'rgba(208, 188, 255, 0.08)',
    textColorBase: '#E6E1E5',
    textColor1: '#E6E1E5',
    textColor2: '#CAC4D0',
    textColor3: '#79747E',
    placeholderColor: '#79747E',
    errorColor: '#F2B8B5',
    errorColorHover: '#F9DEDC',
    errorColorPressed: '#E8A09E',
    errorColorSuppl: '#F2B8B5',
    successColor: '#A8DAB5',
    warningColor: '#FFD699',
    infoColor: '#93C5FD',
    fontSize: '14px',
    borderRadius: '12px',
    borderRadiusSmall: '8px',
    fontFamily: '"Nunito", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
  Card: {
    color: '#2B2930',
    borderColor: '#454349',
    borderRadius: '16px',
    titleTextColor: '#E6E1E5',
    titleFontWeight: '500',
  },
  Input: {
    color: '#1C1B1F',
    borderColor: '#454349',
    borderHoverColor: '#D0BCFF',
    borderFocusColor: '#D0BCFF',
    textColor: '#E6E1E5',
    placeholderColor: '#79747E',
    caretColor: '#D0BCFF',
    colorFocus: '#1C1B1F',
  },
  Button: {
    borderRadiusMedium: '12px',
    borderRadiusSmall: '8px',
    fontWeight: '500',
  },
  Select: {
    peers: {
      InternalSelection: {
        color: '#1C1B1F',
        borderHover: '1px solid #D0BCFF',
        borderFocus: '1px solid #D0BCFF',
        textColor: '#E6E1E5',
        placeholderColor: '#79747E',
      },
    },
  },
  Tag: {
    borderRadius: '8px',
  },
  Scrollbar: {
    color: '#454349',
    colorHover: '#555359',
  },
  Statistic: {
    labelTextColor: '#CAC4D0',
    valueTextColor: '#E6E1E5',
  },
}
</script>
