// ==UserScript==
// @name         bilibili ad block
// @namespace    http://tampermonkey.net/
// @version      1.0.0
// @description  try to take over the world!
// @author       halo
// @match        https://*.bilibili.com/*
// @icon         https://www.google.com/s2/favicons?sz=64&domain=bilibili.com
// @run-at       document-body
// @grant        unsafeWindow
// @grant        GM_addStyle
// @grant        GM_notification
// ==/UserScript==


// 移除搜索框热搜
GM_addStyle(".trending { display: none !important; }");
  // 隐藏游戏中心、下载客户端、赛事和会员购选项
GM_addStyle(`
    a[href*="//game.bilibili.com/platform"] { display: none !important; }
    a[href*="//app.bilibili.com"] { display: none !important; }
    a[href*="//www.bilibili.com/match/home/"] { display: none !important; }
    a[href*="//show.bilibili.com/platform/home.html"] { display: none !important; }
    a[href*="//manga.bilibili.com"] { display: none !important; }
    a[href*="//account.bilibili.com"] { display: none !important; }
`);

// 首页优化
if (location.host === "www.bilibili.com") {
    // 主页轮播图 直播卡片，番剧卡片 ， 客服按钮

     GM_addStyle(".adblock-tips,.recommended-swipe , .floor-single-card , .palette-button-wrap { display: none !important; }");
    // 移除推广卡片
    // 监听 DOM 变化以处理动态加载的内容
    const observer = new MutationObserver(() => {
    document.querySelectorAll('.bili-video-card.is-rcmd:not(.enable-no-interest),.recommended-swipe ').forEach(el => {
        el.style.display = 'none';
    });
    });

// 开始监听
observer.observe(document.body, { childList: true, subtree: true });
}

// 视频页
if (location.href.startsWith('https://www.bilibili.com/video/')) {
     GM_addStyle(".video-page-operator-card-small,#bar ,  .pop-live-small-mode,.slide-ad-exp, .video-page-game-card-small, .activity-m-v1 , .act-now ,.video-page-special-card-small{ display: none !important; }");
}

// 直播页
if (location.href.startsWith('https://live.bilibili.com/')) {
     GM_addStyle("#chat-control-panel-vm {  background-color: rgb(215,236,249); background-image: none!important;}" )
     GM_addStyle("#gift-control-vm , .side-bar-cntr, .room-info-ctnr , .flip-view ,.right-ctnr , #rank-list-vm , .medal-section { display: none !important; }");
}




