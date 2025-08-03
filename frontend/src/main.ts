// Global variables
let currentUser: any = null;

// Types
interface LoginResult {
    success: boolean;
    message: string;
    user?: {
        userId: string;
        platformKeys: PlatformKey[];
    };
    userId?: string;
    platformKeys?: PlatformKey[];
}

interface PlatformKey {
    platformName: string;
    name: string;
    platformAccessKey: string;
    platformSecretKey: string;
}

interface UserData {
    userId: string;
    platformKeys: PlatformKey[];
}

// Wails API 호출을 위한 함수들
async function callLogin(userID: string, password: string): Promise<LoginResult> {
    try {
        const { Login } = await import('../wailsjs/go/main/App');
        const result = await Login(userID, password);
        
        // 결과가 없거나 undefined인 경우 처리
        if (!result) {
            return { success: false, message: '로그인 중 오류가 발생했습니다.' };
        }
        
        return result as LoginResult;
    } catch (error) {
        console.error('Login error:', error);
        return { success: false, message: '로그인 중 오류가 발생했습니다.' };
    }
}

async function callRegister(userID: string, password: string): Promise<LoginResult> {
    try {
        const { Register } = await import('../wailsjs/go/main/App');
        const result = await Register(userID, password);
        return result as LoginResult;
    } catch (error) {
        console.error('Register error:', error);
        return { success: false, message: '회원가입 중 오류가 발생했습니다.' };
    }
}

async function callGetUserInfo(userID: string): Promise<LoginResult> {
    try {
        const { GetAccountInfo } = await import('../wailsjs/go/main/App');
        const result = await GetAccountInfo(userID);
        return result as LoginResult;
    } catch (error) {
        console.error('GetUserInfo error:', error);
        return { success: false, message: '사용자 정보를 불러오는 중 오류가 발생했습니다.' };
    }
}

// 플랫폼 관련 API 호출 함수들
async function callGetPlatformInfo(userID: string): Promise<any> {
    try {
        const { GetPlatformInfo } = await import('../wailsjs/go/main/App');
        const result = await GetPlatformInfo(userID);
        return result;
    } catch (error) {
        console.error('GetPlatformInfo error:', error);
        return { success: false, message: '플랫폼 정보를 불러오는 중 오류가 발생했습니다.' };
    }
}

async function callAddPlatform(userID: string, platform: string, name: string, accessKey: string, secretKey: string): Promise<any> {
    try {
        const { AddPlatform } = await import('../wailsjs/go/main/App');
        const result = await AddPlatform(userID, platform, name, accessKey, secretKey);
        return result;
    } catch (error) {
        console.error('AddPlatform error:', error);
        return { success: false, message: '플랫폼 추가 중 오류가 발생했습니다.' };
    }
}

async function callUpdatePlatform(userID: string, oldPlatform: string, oldName: string, newPlatform: string, newName: string, accessKey: string, secretKey: string): Promise<any> {
    try {
        const { UpdatePlatform } = await import('../wailsjs/go/main/App');
        const result = await UpdatePlatform(userID, oldPlatform, oldName, newPlatform, newName, accessKey, secretKey);
        return result;
    } catch (error) {
        console.error('UpdatePlatform error:', error);
        return { success: false, message: '플랫폼 수정 중 오류가 발생했습니다.' };
    }
}

async function callRemovePlatform(userID: string, platform: string, name: string): Promise<any> {
    try {
        const { RemovePlatform } = await import('../wailsjs/go/main/App');
        const result = await RemovePlatform(userID, platform, name);
        return result;
    } catch (error) {
        console.error('RemovePlatform error:', error);
        return { success: false, message: '플랫폼 제거 중 오류가 발생했습니다.' };
    }
}

async function callGetAllPlatforms(): Promise<any> {
    try {
        const { GetAllPlatforms } = await import('../wailsjs/go/main/App');
        const result = await GetAllPlatforms();
        return result;
    } catch (error) {
        console.error('GetAllPlatforms error:', error);
        return { success: false, message: '플랫폼 목록을 불러오는 중 오류가 발생했습니다.' };
    }
}

// Utility functions
function showAlert(message: string, type: 'success' | 'error' = 'success'): void {
    // Wails에서는 alert를 사용하지 않고 console.log로 대체
    console.log(`[${type.toUpperCase()}] ${message}`);
    
    // 에러인 경우 alert로도 표시 (더 오래 보이도록)
    if (type === 'error') {
        alert(`[ERROR] ${message}`);
    }
}

function showLoading(element: HTMLButtonElement): void {
    element.disabled = true;
    element.innerHTML = '<span class="loading-spinner"></span> 처리 중...';
}

function hideLoading(element: HTMLButtonElement, originalText: string): void {
    element.disabled = false;
    element.textContent = originalText;
}

// Page navigation
function showPage(pageName: string): void {
    // Hide all pages
    const pages = document.querySelectorAll('.page');
    pages.forEach(page => {
        page.classList.remove('active');
    });

    // Show target page
    const targetPage = document.getElementById(`${pageName}-page`);
    if (targetPage) {
        targetPage.classList.add('active');
    }
}

// Form validation
function validateForm(formData: { userID: string; password: string; confirmPassword?: string }): string[] {
    const errors: string[] = [];

    if (!formData.userID || formData.userID.trim() === '') {
        errors.push('사용자 ID를 입력해주세요.');
    }

    if (!formData.password || formData.password.trim() === '') {
        errors.push('비밀번호를 입력해주세요.');
    }

    if (formData.confirmPassword && formData.password !== formData.confirmPassword) {
        errors.push('비밀번호가 일치하지 않습니다.');
    }

    return errors;
}

// Login form handler
async function handleLogin(event: Event): Promise<void> {
    event.preventDefault();

    const form = event.target as HTMLFormElement;
    const submitBtn = form.querySelector('button[type="submit"]') as HTMLButtonElement;
    const originalText = submitBtn.textContent || '';

    // Get form data
    const formData = {
        userID: (form.userID as HTMLInputElement).value.trim(),
        password: (form.password as HTMLInputElement).value
    };

    // Validate form
    const errors = validateForm(formData);
    if (errors.length > 0) {
        showAlert(errors.join('\n'), 'error');
        return;
    }

    // Show loading
    showLoading(submitBtn);

    // Call login API
    try {
        const result = await callLogin(formData.userID, formData.password);
        
        if (result && result.success) {
            // Store user info
            currentUser = result.user;
            localStorage.setItem('userID', formData.userID);
            
            showAlert(result.message, 'success');
            
            // Redirect to dashboard
            setTimeout(() => {
                showPage('dashboard');
                getUserInfo(formData.userID);
            }, 1000);
        } else {
            // 에러 메시지 처리 개선
            let errorMessage = '로그인에 실패했습니다.';
            if (result && result.message) {
                errorMessage = result.message;
            } else if (result && !result.success) {
                errorMessage = '아이디를 찾을 수 없거나 비밀번호가 잘못되었습니다.';
            }
            showAlert(errorMessage, 'error');
        }
    } catch (error) {
        showAlert('로그인 중 오류가 발생했습니다.', 'error');
    } finally {
        hideLoading(submitBtn, originalText);
    }
}

// Register form handler
async function handleRegister(event: Event): Promise<void> {
    event.preventDefault();

    const form = event.target as HTMLFormElement;
    const submitBtn = form.querySelector('button[type="submit"]') as HTMLButtonElement;
    const originalText = submitBtn.textContent || '';

    // Get form data
    const formData = {
        userID: (form.userID as HTMLInputElement).value.trim(),
        password: (form.password as HTMLInputElement).value,
        confirmPassword: (form.confirmPassword as HTMLInputElement).value
    };

    // Validate form
    const errors = validateForm(formData);
    if (errors.length > 0) {
        showAlert(errors.join('\n'), 'error');
        return;
    }

    // Show loading
    showLoading(submitBtn);

    // Call register API
    try {
        const result = await callRegister(formData.userID, formData.password);
        
        if (result.success) {
            showAlert(result.message, 'success');
            
            // Redirect to login
            setTimeout(() => {
                showPage('login');
                // 폼 초기화
                form.reset();
            }, 1000);
        } else {
            showAlert(result.message, 'error');
        }
    } catch (error) {
        showAlert('회원가입 중 오류가 발생했습니다.', 'error');
    } finally {
        hideLoading(submitBtn, originalText);
    }
}

// Get user info
async function getUserInfo(userID: string): Promise<void> {
    try {
        const result = await callGetUserInfo(userID);
        
        if (result.success) {
            displayUserInfo(result as UserData);
        } else {
            showAlert(result.message, 'error');
        }
    } catch (error) {
        showAlert('사용자 정보를 불러오는 중 오류가 발생했습니다.', 'error');
    }
}

// Display user info in dashboard
function displayUserInfo(data: UserData): void {
    // 사용자 정보 업데이트
    const currentUserSpan = document.getElementById('current-user');
    if (currentUserSpan) {
        currentUserSpan.textContent = data.userId;
    }

    // 플랫폼 목록 업데이트
    const platformsContent = document.getElementById('platforms-content');

    if (platformsContent && data.platformKeys) {
        // 현재 플랫폼 목록 저장
        currentPlatforms = data.platformKeys;
        
        platformsContent.innerHTML = '';
        
        if (data.platformKeys.length === 0) {
            platformsContent.innerHTML = '<p>연결된 플랫폼이 없습니다. "추가" 버튼을 클릭하여 플랫폼을 추가하세요.</p>';
        } else {
            const platformList = document.createElement('div');
            platformList.className = 'platform-list';
            
            data.platformKeys.forEach((platform: PlatformKey, index: number) => {
                const platformItem = document.createElement('div');
                platformItem.className = 'platform-item';
                platformItem.innerHTML = `
                    <div class="platform-item-header">
                        <div class="platform-item-title">
                            <span class="platform-name">${platform.platformName}</span>
                            <span class="platform-alias">${platform.name}</span>
                        </div>
                        <div class="platform-actions">
                            <button class="btn btn-secondary" onclick="showEditPlatformModal('${platform.platformName}', '${platform.name}', '${platform.platformAccessKey}', '${platform.platformSecretKey}')">
                                수정
                            </button>
                            <button class="btn btn-secondary" onclick="removePlatform('${platform.platformName}', '${platform.name}')">
                                삭제
                            </button>
                        </div>
                    </div>
                    <div class="platform-details">
                        <div class="platform-detail">
                            <div class="platform-detail-label">Access Key</div>
                            <div class="platform-detail-value">${platform.platformAccessKey}</div>
                        </div>
                        <div class="platform-detail">
                            <div class="platform-detail-label">Secret Key</div>
                            <div class="platform-detail-value">${platform.platformSecretKey}</div>
                        </div>
                    </div>
                `;
                platformList.appendChild(platformItem);
            });
            
            platformsContent.appendChild(platformList);
        }
    }
}

// Dashboard tab navigation
function showDashboardTab(tabName: string): void {
    // Hide all tabs
    const tabs = document.querySelectorAll('.dashboard-tab');
    tabs.forEach(tab => {
        tab.classList.remove('active');
    });

    // Remove active class from all nav items
    const navItems = document.querySelectorAll('.sidebar-nav li');
    navItems.forEach(item => {
        item.classList.remove('active');
    });

    // Show target tab
    const targetTab = document.getElementById(`${tabName}-tab`);
    if (targetTab) {
        targetTab.classList.add('active');
    }

    // Add active class to nav item
    const targetNavItem = document.querySelector(`[onclick="showDashboardTab('${tabName}')"]`)?.parentElement;
    if (targetNavItem) {
        targetNavItem.classList.add('active');
    }

    // Update dashboard title
    const dashboardTitle = document.getElementById('dashboard-title');
    if (dashboardTitle) {
        const titles: { [key: string]: string } = {
            'platforms': '거래소 Key 관리',
            'trading': '예약 매도 관리',
            'settings': '설정',
            'system-log': '시스템 로그',
            'log-viewer-tabs': '로그 뷰어 (탭 기반)',
            'log-viewer-grid': '로그 뷰어 (그리드 레이아웃)',
            'log-viewer-dropdown': '로그 뷰어 (드롭다운 선택기)'
        };
        dashboardTitle.textContent = titles[tabName] || '대시보드';
    }
    

    
                        // 로그 뷰어 그리드가 활성화되면 초기화
            if (tabName === 'log-viewer-grid') {
                // 그리드 기반 로그 뷰어 초기화
                initializeGridLogs();
            }
            
            // 예약 매도 관리 탭이 활성화되면 목록 로드
            if (tabName === 'trading') {
                const userID = localStorage.getItem('userID');
                if (userID) {
                    loadSellOrders(userID);
                    // 웹소켓 연결 시작
                    startWebSocketConnection(userID);
                }
            }
}

// Platform management functions
let currentEditingPlatform: any = null;

function showAddPlatformModal(): void {
    currentEditingPlatform = null;
    document.getElementById('modal-title')!.textContent = '플랫폼 추가';
    document.getElementById('platform-submit-btn')!.textContent = '추가';
    (document.getElementById('platform-form') as HTMLFormElement).reset();
    document.getElementById('platform-modal')!.classList.add('active');
}

function showEditPlatformModal(platformName: string, name: string, accessKey: string, secretKey: string): void {
    // HTML 엔티티 디코딩
    const decodedPlatformName = platformName.replace(/&apos;/g, "'");
    const decodedName = name.replace(/&apos;/g, "'");
    const decodedAccessKey = accessKey.replace(/&apos;/g, "'");
    const decodedSecretKey = secretKey.replace(/&apos;/g, "'");
    
    currentEditingPlatform = { 
        platformName: decodedPlatformName, 
        name: decodedName, 
        platformAccessKey: decodedAccessKey, 
        platformSecretKey: decodedSecretKey 
    };
    document.getElementById('modal-title')!.textContent = '플랫폼 수정';
    document.getElementById('platform-submit-btn')!.textContent = '수정';
    
    // 폼에 기존 값들 채우기
    const form = document.getElementById('platform-form') as HTMLFormElement;
    (form.querySelector('[name="platformName"]') as HTMLSelectElement).value = decodedPlatformName;
    (form.querySelector('[name="name"]') as HTMLInputElement).value = decodedName;
    (form.querySelector('[name="accessKey"]') as HTMLInputElement).value = decodedAccessKey;
    (form.querySelector('[name="secretKey"]') as HTMLInputElement).value = decodedSecretKey;
    
    document.getElementById('platform-modal')!.classList.add('active');
}

function closePlatformModal(): void {
    document.getElementById('platform-modal')!.classList.remove('active');
    currentEditingPlatform = null;
}

async function handlePlatformSubmit(event: Event): Promise<void> {
    event.preventDefault();
    
    const form = event.target as HTMLFormElement;
    const submitBtn = form.querySelector('button[type="submit"]') as HTMLButtonElement;
    const originalText = submitBtn.textContent || '';
    
    // 폼 데이터 가져오기
    const formData = new FormData(form);
    const platformName = formData.get('platformName') as string;
    const name = formData.get('name') as string;
    const accessKey = formData.get('accessKey') as string;
    const secretKey = formData.get('secretKey') as string;
    
    // 입력값 검증
    if (!platformName || !name || !accessKey || !secretKey) {
        showAlert('모든 필드를 입력해주세요.', 'error');
        return;
    }
    
    // 로딩 표시
    showLoading(submitBtn);
    
    try {
        const userID = localStorage.getItem('userID');
        if (!userID) {
            showAlert('로그인이 필요합니다.', 'error');
            return;
        }
        
        let result;
        if (currentEditingPlatform) {
            // 수정 모드 - 기존 정보와 새 정보를 모두 전달
            result = await callUpdatePlatform(
                userID, 
                currentEditingPlatform.platformName,  // 기존 플랫폼명
                currentEditingPlatform.name,         // 기존 별칭
                platformName,                        // 새 플랫폼명
                name,                                // 새 별칭
                accessKey, 
                secretKey
            );
        } else {
            // 추가 모드
            result = await callAddPlatform(userID, platformName, name, accessKey, secretKey);
        }
        
        if (result.success) {
            showAlert(result.message, 'success');
            closePlatformModal();
            // 플랫폼 목록 새로고침
            getUserInfo(userID);
        } else {
            showAlert(result.message, 'error');
        }
    } catch (error) {
        showAlert('플랫폼 처리 중 오류가 발생했습니다.', 'error');
    } finally {
        hideLoading(submitBtn, originalText);
    }
}

async function removePlatform(platformName: string, name: string): Promise<void> {
    // 특수문자 처리
    const decodedPlatformName = platformName.replace(/&apos;/g, "'").replace(/&quot;/g, '"');
    const decodedName = name.replace(/&apos;/g, "'").replace(/&quot;/g, '"');
    
    try {
        const userID = localStorage.getItem('userID');
        
        if (!userID) {
            showAlert('로그인이 필요합니다.', 'error');
            return;
        }
        
        const result = await callRemovePlatform(userID, decodedPlatformName, decodedName);
        
        if (result.success) {
            showAlert(result.message, 'success');
            getUserInfo(userID);
        } else {
            showAlert(result.message, 'error');
        }
    } catch (error) {
        showAlert(`삭제 중 오류 발생: ${error}`, 'error');
    }
}

// Add order function
function addSellOrder(event?: Event): void {
    // 탭 이동을 완전히 막기
    if (event) {
        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }
    // API Key 목록을 드롭다운에 로드
    loadPlatformKeys();
    document.getElementById('sell-order-modal')!.classList.add('active');
}

function closeSellOrderModal(): void {
    document.getElementById('sell-order-modal')!.classList.remove('active');
    (document.getElementById('sell-order-form') as HTMLFormElement).reset();
}

function loadPlatformKeys(): void {
    const userID = localStorage.getItem('userID');
    
    if (!userID) {
        showAlert('로그인이 필요합니다.', 'error');
        return;
    }

    // 사용자 정보를 가져와서 API Key 목록을 드롭다운에 추가
    getUserInfo(userID).then(() => {
        const select = document.getElementById('platformKey-select') as HTMLSelectElement;
        select.innerHTML = '<option value="">API Key를 선택하세요</option>';
        
        // 현재 플랫폼 목록을 드롭다운에 추가
        currentPlatforms.forEach(platform => {
            const option = document.createElement('option');
            option.value = `${platform.platformName}-${platform.name}`;
            option.textContent = `${platform.platformName} - ${platform.name}`;
            select.appendChild(option);
        });
    });
}

function toggleOrderDetail(orderId: string): void {
    const detailElement = document.getElementById(`${orderId}-detail`);
    const orderHeader = detailElement?.previousElementSibling as HTMLElement;
    const expandIcon = orderHeader?.querySelector('.expand-icon') as HTMLElement;
    
    if (detailElement) {
        const isExpanded = detailElement.classList.contains('expanded');
        
        if (isExpanded) {
            detailElement.classList.remove('expanded');
            if (expandIcon) {
                expandIcon.style.transform = 'rotate(0deg)';
            }
        } else {
            detailElement.classList.add('expanded');
            if (expandIcon) {
                expandIcon.style.transform = 'rotate(90deg)';
            }
        }
    }
}

async function handleSellOrderSubmit(event: Event): Promise<void> {
    console.log('=== handleSellOrderSubmit 호출됨 ===');
    event.preventDefault();
    
    const form = event.target as HTMLFormElement;
    const submitBtn = form.querySelector('button[type="submit"]') as HTMLButtonElement;
    const originalText = submitBtn.textContent || '';
    
    // 폼 데이터 가져오기
    const formData = new FormData(form);
    const platformKey = formData.get('platformKey') as string;
    const orderName = formData.get('orderName') as string;
    const symbol = formData.get('symbol') as string;
    const price = parseFloat(formData.get('price') as string);
    const quantity = parseFloat(formData.get('quantity') as string);
    const term = parseFloat(formData.get('term') as string);
    
    console.log('폼 데이터:', {
        platformKey,
        orderName,
        symbol,
        price,
        quantity,
        term
    });
    
    // 입력값 검증
    console.log('입력값 검증 시작');
    if (!platformKey || !orderName || !symbol || isNaN(price) || isNaN(quantity) || isNaN(term)) {
        console.log('입력값 검증 실패:', { platformKey, orderName, symbol, price, quantity, term });
        showAlert('모든 필드를 올바르게 입력해주세요.', 'error');
        return;
    }
    console.log('입력값 검증 통과');
    
    // 로딩 표시
    showLoading(submitBtn);
    
    try {
        const userID = localStorage.getItem('userID');
        if (!userID) {
            showAlert('로그인이 필요합니다.', 'error');
            return;
        }
        
        // 플랫폼 정보 파싱
        const [platformName, platformNickName] = platformKey.split('-');
        
        // 예약 매도 주문 추가
        const result = await callAddSellOrder(userID, orderName, symbol, price, quantity, term, platformName, platformNickName);
        
        if (result.success) {
            showAlert(result.message, 'success');
            closeSellOrderModal();
            // 예약 매도 목록 새로고침
            await loadSellOrders(userID);
        } else {
            showAlert(result.message, 'error');
        }
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('handleSellOrderSubmit error:', error);
        showAlert('예약 매도 추가 중 오류가 발생했습니다.', 'error');
    } finally {
        hideLoading(submitBtn, originalText);
    }
}

async function callAddSellOrder(userID: string, orderName: string, symbol: string, price: number, quantity: number, term: number, platformName: string, platformNickName: string): Promise<any> {
    try {
        const { AddSellOrder } = await import('../wailsjs/go/main/App');
        const result = await AddSellOrder(userID, orderName, symbol, price, quantity, term, platformName, platformNickName);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('AddSellOrder error:', error);
        return { success: false, message: `예약 매도 추가 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 예약 매도 데이터 타입 정의
interface SellOrder {
    id: string;
    name: string;
    symbol: string;
    price: number;
    quantity: number;
    term: number;
    platform: string;
    platformNickName: string;
    status: 'active' | 'inactive';
}

// 예약 매도 데이터 (실제 데이터로 대체 예정)
const sellOrders: SellOrder[] = [];

async function loadSellOrders(userID: string): Promise<void> {
    try {
        const result = await callGetSellOrders(userID);
        if (result.success) {
            // sellOrders 배열을 실제 데이터로 업데이트
            sellOrders.length = 0; // 배열 초기화
            if (result.sellOrders && Array.isArray(result.sellOrders)) {
                result.sellOrders.forEach((order: any) => {
                    sellOrders.push({
                        id: order.name, // name을 id로 사용
                        name: order.name,
                        symbol: order.symbol,
                        price: order.price,
                        quantity: order.quantity,
                        term: order.term,
                        platform: order.platform,
                        platformNickName: order.platformNickName,
                        status: 'active'
                    });
                });
            }
            renderSellOrders();
        } else {
            console.error('예약 매도 목록 로드 실패:', result.message);
        }
    } catch (error) {
        console.error('예약 매도 목록 로드 중 오류:', error);
    }
}

async function callGetSellOrders(userID: string): Promise<any> {
    try {
        const { GetSellOrders } = await import('../wailsjs/go/main/App');
        const result = await GetSellOrders(userID);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('GetSellOrders error:', error);
        return { success: false, message: `예약 매도 목록 조회 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 워커 로그 스트리밍 API 호출
async function callGetWorkerLogsStream(userID: string): Promise<any> {
    try {
        const { GetWorkerLogsStream } = await import('../wailsjs/go/main/App');
        const result = await GetWorkerLogsStream(userID);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('GetWorkerLogsStream error:', error);
        return { success: false, message: `워커 로그 스트리밍 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 특정 주문의 로그 조회 API 호출
async function callGetOrderLogs(userID: string, orderName: string): Promise<any> {
    try {
        const { GetOrderLogs } = await import('../wailsjs/go/main/App');
        const result = await GetOrderLogs(userID, orderName);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('GetOrderLogs error:', error);
        return { success: false, message: `주문 로그 조회 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 로그 구독 API 호출
async function callSubscribeToLogs(userID: string): Promise<any> {
    try {
        const { SubscribeToLogs } = await import('../wailsjs/go/main/App');
        const result = await SubscribeToLogs(userID);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('SubscribeToLogs error:', error);
        return { success: false, message: `로그 구독 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 통합된 로그 조회 API 호출
async function callGetUnifiedLogs(userID: string): Promise<any> {
    try {
        const { GetUnifiedLogs } = await import('../wailsjs/go/main/App');
        const result = await GetUnifiedLogs(userID);
        return result;
    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : '알 수 없는 오류가 발생했습니다.';
        console.error('GetUnifiedLogs error:', error);
        return { success: false, message: `통합 로그 조회 중 오류가 발생했습니다: ${errorMessage}` };
    }
}

// 예약 매도 목록 렌더링
function renderSellOrders(): void {
    const container = document.getElementById('sell-order-list-container');
    if (!container) return;
    
    container.innerHTML = '';
    
    if (sellOrders.length === 0) {
        container.innerHTML = '<div class="no-orders">등록된 예약 매도가 없습니다.</div>';
        return;
    }
    
    sellOrders.forEach(order => {
        const orderElement = createSellOrderElement(order);
        container.appendChild(orderElement);
    });
}

// 실시간 로그 업데이트 시작
let logUpdateInterval: NodeJS.Timeout | null = null;

function startRealTimeLogUpdates(): void {
    const userID = localStorage.getItem('userID');
    if (!userID) return;
    
    // 기존 인터벌이 있으면 제거
    if (logUpdateInterval) {
        clearInterval(logUpdateInterval);
    }
    
    console.log('실시간 로그 업데이트 시작');
    
    // 5초마다 로그 업데이트
    logUpdateInterval = setInterval(async () => {
        console.log('로그 업데이트 실행');
        await updateAllOrderLogs(userID);
    }, 5000);
}

// 모든 주문의 로그 업데이트
async function updateAllOrderLogs(userID: string): Promise<void> {
    for (const order of sellOrders) {
        await updateOrderLogs(userID, order.id);
    }
}

// 웹소켓 연결을 위한 변수들
let wsConnection: WebSocket | null = null;
let isConnected = false;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;

// 웹소켓 연결 시작
async function startWebSocketConnection(userID: string): Promise<void> {
    try {
        console.log('웹소켓 연결 시작:', userID);
        
        // 웹소켓 연결 생성
        const wsUrl = `ws://localhost:8080/ws?userId=${encodeURIComponent(userID)}`;
        wsConnection = new WebSocket(wsUrl);
        
        wsConnection.onopen = () => {
            console.log('웹소켓 연결 성공');
            isConnected = true;
            reconnectAttempts = 0;
        };
        
        wsConnection.onmessage = (event) => {
            try {
                const log = JSON.parse(event.data);
                console.log('웹소켓 로그 수신:', log);
                processUnifiedLog(log);
            } catch (error) {
                console.error('웹소켓 메시지 파싱 실패:', error);
            }
        };
        
        wsConnection.onclose = () => {
            console.log('웹소켓 연결 종료');
            isConnected = false;
            
            // 재연결 시도
            if (reconnectAttempts < maxReconnectAttempts) {
                reconnectAttempts++;
                console.log(`웹소켓 재연결 시도 ${reconnectAttempts}/${maxReconnectAttempts}`);
                setTimeout(() => {
                    startWebSocketConnection(userID);
                }, 5000);
            }
        };
        
        wsConnection.onerror = (error) => {
            console.error('웹소켓 연결 오류:', error);
        };
        
    } catch (error) {
        console.error('웹소켓 연결 실패:', error);
    }
}

// 웹소켓 연결 종료
function stopWebSocketConnection(): void {
    if (wsConnection) {
        wsConnection.close();
        wsConnection = null;
    }
    isConnected = false;
}

// 통합된 로그 처리 (배열)
function processUnifiedLogs(logs: any[]): void {
    logs.forEach(log => {
        processUnifiedLog(log);
    });
}

// 통합된 로그 처리 (단일)
function processUnifiedLog(log: any): void {
    // 플랫폼과 별칭으로 해당 주문 찾기
    const order = sellOrders.find(o => 
        o.platform.toLowerCase() === log.platform.toLowerCase() && 
        o.platformNickName === log.nickname
    );
    
    if (order) {
        // 해당 주문의 로그 컨테이너에 로그 추가
        addLogToOrder(order.id, log);
    }
}

// 주문에 로그 추가
function addLogToOrder(orderName: string, log: any): void {
    const logContainer = document.getElementById(`${orderName}-logs`);
    if (!logContainer) return;
    
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${log.logType || 'info'}`;
    
    const timestamp = new Date(log.timestamp).toLocaleTimeString();
    const message = log.message;
    
    logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
    logContainer.appendChild(logEntry);
    
    // 자동 스크롤
    logContainer.scrollTop = logContainer.scrollHeight;
    
    // 최대 20개 로그만 유지
    const logEntries = logContainer.querySelectorAll('.log-entry');
    if (logEntries.length > 20) {
        logEntries[0].remove();
    }
}

// 특정 주문의 로그 업데이트
async function updateOrderLogs(userID: string, orderName: string): Promise<void> {
    try {
        console.log('로그 업데이트 시작:', userID, orderName);
        const result = await callGetOrderLogs(userID, orderName);
        console.log('로그 업데이트 결과:', result);
        if (result.success && result.logs) {
            console.log('로그 업데이트 표시:', result.logs.length, '개');
            displayOrderLogs(orderName, result.logs);
        } else {
            console.log('로그 업데이트 없음:', result);
        }
    } catch (error) {
        console.error('로그 업데이트 실패:', error);
    }
}

// 주문 로그 로드
async function loadOrderLogs(userID: string, orderName: string): Promise<void> {
    try {
        console.log('로그 로드 시작:', userID, orderName);
        const result = await callGetOrderLogs(userID, orderName);
        console.log('로그 로드 결과:', result);
        if (result.success && result.logs) {
            console.log('로그 표시 시작:', result.logs.length, '개');
            displayOrderLogs(orderName, result.logs);
        } else {
            console.log('로그 없음 또는 실패:', result);
        }
    } catch (error) {
        console.error('로그 로드 실패:', error);
    }
}

// 주문 로그 표시
function displayOrderLogs(orderName: string, logs: any[]): void {
    const logContainer = document.getElementById(`${orderName}-logs`);
    if (!logContainer) {
        console.log('로그 컨테이너를 찾을 수 없습니다:', orderName);
        return;
    }
    
    console.log('로그 표시 시작:', orderName, logs.length, '개');
    
    // 기존 로그 제거
    logContainer.innerHTML = '';
    
    // 최근 10개 로그만 표시
    const recentLogs = logs.slice(-10);
    
    recentLogs.forEach((log, index) => {
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        
        const timestamp = new Date(log.timestamp).toLocaleTimeString();
        const message = log.message;
        
        logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
        logContainer.appendChild(logEntry);
        
        console.log(`로그 ${index + 1}: [${timestamp}] ${message}`);
    });
    
    // 자동 스크롤
    logContainer.scrollTop = logContainer.scrollHeight;
    
    console.log('로그 표시 완료:', orderName);
}

// 모달 로그 로드
async function loadModalLogs(userID: string, orderName: string): Promise<void> {
    try {
        const result = await callGetOrderLogs(userID, orderName);
        if (result.success && result.logs) {
            displayModalLogs(result.logs);
        }
    } catch (error) {
        console.error('모달 로그 로드 실패:', error);
    }
}

// 모달 로그 표시
function displayModalLogs(logs: any[]): void {
    const logContent = document.getElementById('log-content');
    if (!logContent) return;
    
    // 기존 로그 제거
    logContent.innerHTML = '';
    
    // 최근 20개 로그 표시
    const recentLogs = logs.slice(-20);
    
    recentLogs.forEach(log => {
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${log.logType || 'info'}`;
        
        const timestamp = new Date(log.timestamp).toLocaleTimeString();
        const message = log.message;
        
        logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
        logContent.appendChild(logEntry);
    });
    
    // 자동 스크롤
    logContent.scrollTop = logContent.scrollHeight;
}

// 예약 매도 요소 생성
function createSellOrderElement(order: SellOrder): HTMLElement {
    const orderItem = document.createElement('div');
    orderItem.className = 'sell-order-item';
    
    const header = document.createElement('div');
    header.className = 'order-header';
    header.onclick = () => toggleOrderDetail(order.id);
    
    const expandIcon = document.createElement('div');
    expandIcon.className = 'expand-icon';
    expandIcon.textContent = '>';
    
    const orderInfo = document.createElement('div');
    orderInfo.className = 'order-info';
    
    const platform = document.createElement('div');
    platform.className = 'order-platform';
    platform.textContent = order.platform;
    
    const symbol = document.createElement('div');
    symbol.className = 'order-symbol';
    symbol.textContent = order.symbol;
    
    const price = document.createElement('div');
    price.className = 'order-price';
    price.textContent = formatPrice(order.price, order.symbol);
    
    const quantity = document.createElement('div');
    quantity.className = 'order-quantity';
    quantity.textContent = order.quantity.toString();
    
    const term = document.createElement('div');
    term.className = 'order-term';
    term.textContent = `${order.term}초`;
    
    orderInfo.appendChild(platform);
    orderInfo.appendChild(symbol);
    orderInfo.appendChild(price);
    orderInfo.appendChild(quantity);
    orderInfo.appendChild(term);
    
    const orderActions = document.createElement('div');
    orderActions.className = 'order-actions';
    
    const logButton = document.createElement('button');
    logButton.className = 'btn btn-secondary';
    logButton.textContent = '로그 보기';
    logButton.onclick = (event) => viewInModal(order.id, event);
    
    orderActions.appendChild(logButton);
    
    header.appendChild(expandIcon);
    header.appendChild(orderInfo);
    header.appendChild(orderActions);
    
    const detail = document.createElement('div');
    detail.className = 'order-detail';
    detail.id = `${order.id}-detail`;
    
    const detailContent = document.createElement('div');
    detailContent.className = 'detail-content';
    
    const detailTitle = document.createElement('h4');
    detailTitle.textContent = '예약 매도 상세 정보';
    
    const detailGrid = document.createElement('div');
    detailGrid.className = 'detail-grid';
    
    // 상세 정보 항목들 생성
    const detailItems = [
        { label: '별칭:', value: order.name },
        { label: '플랫폼:', value: order.platform },
        { label: '심볼:', value: order.symbol },
        { label: '목표가:', value: formatPrice(order.price, order.symbol) },
        { label: '수량:', value: order.quantity.toString() },
        { label: '주기:', value: `${order.term}초` }
    ];
    
    detailItems.forEach(item => {
        const detailItem = document.createElement('div');
        detailItem.className = 'detail-item';
        
        const label = document.createElement('span');
        label.className = 'label';
        label.textContent = item.label;
        
        const value = document.createElement('span');
        value.className = 'value';
        value.textContent = item.value;
        
        detailItem.appendChild(label);
        detailItem.appendChild(value);
        detailGrid.appendChild(detailItem);
    });
    
    detailContent.appendChild(detailTitle);
    detailContent.appendChild(detailGrid);
    
    // 로그 창 추가
    const orderLogs = document.createElement('div');
    orderLogs.className = 'order-logs';
    
    const logsTitle = document.createElement('h5');
    logsTitle.textContent = '실시간 로그';
    
    const logContent = document.createElement('div');
    logContent.className = 'log-content';
    logContent.id = `${order.id}-logs`;
    
    // 초기 로그 로드
    const userID = localStorage.getItem('userID');
    if (userID) {
        loadOrderLogs(userID, order.id);
    }
    
    orderLogs.appendChild(logsTitle);
    orderLogs.appendChild(logContent);
    
    // 상세 정보 제거하고 로그만 표시
    detail.appendChild(orderLogs);
    
    orderItem.appendChild(header);
    orderItem.appendChild(detail);
    
    return orderItem;
}

// 가격 포맷팅 함수
function formatPrice(price: number, symbol: string): string {
    if (symbol.includes('KRW')) {
        return `₩${price.toLocaleString()}`;
    } else if (symbol.includes('USDT')) {
        return `$${price.toLocaleString()}`;
    }
    return price.toString();
}



function viewInModal(orderId: string, event?: Event): void {
    // 이벤트 전파 중단
    if (event) {
        event.stopPropagation();
    }
    
    // sellOrders 배열에서 해당 주문 찾기
    const order = sellOrders.find(o => o.id === orderId);
    if (order) {
        // 모달 제목 설정
        const modalTitle = document.getElementById('log-modal-title');
        if (modalTitle) {
            modalTitle.textContent = `${order.name} (${order.platform}) 로그`;
        }
        
        // 로그 내용 생성
        const logContent = document.getElementById('log-content');
        if (logContent) {
            logContent.innerHTML = '';
            
            // 실제 로그 로드
            const userID = localStorage.getItem('userID');
            if (userID) {
                loadModalLogs(userID, orderId);
            }
        }
        
        // 모달 표시
        const modal = document.getElementById('log-modal');
        if (modal) {
            modal.classList.add('active');
        }
    }
}

function closeLogModal(): void {
    document.getElementById('log-modal')!.classList.remove('active');
}

// 로그 데이터 정의 (실제 데이터로 대체 예정)
const logData: { [key: string]: { title: string; logs: Array<{ time: string; message: string; type: string }> } } = {};



// 로그 내용 표시 함수
function displayLogContent(orderId: string, containerId: string): void {
    const order = logData[orderId as keyof typeof logData];
    const container = document.getElementById(containerId);
    
    if (order && container) {
        container.innerHTML = '';
        order.logs.forEach(log => {
            const logEntry = document.createElement('div');
            logEntry.className = `log-entry ${log.type}`;
            logEntry.innerHTML = `<span class="timestamp">[${log.time}]</span> ${log.message}`;
            container.appendChild(logEntry);
        });
    }
}

// 그리드 로그 패널 초기화
function initializeGridLogs(): void {
    const container = document.getElementById('log-grid-container');
    if (!container) return;
    
    container.innerHTML = '';
    
    Object.keys(logData).forEach((orderId, index) => {
        const order = logData[orderId];
        if (!order) return;
        
        const panel = createLogPanel(orderId, order.title, 'active');
        container.appendChild(panel);
        displayLogContent(orderId, `grid-log-${index + 1}`);
    });
}

// 로그 패널 생성 함수
function createLogPanel(orderId: string, title: string, status: 'active' | 'inactive'): HTMLElement {
    const panel = document.createElement('div');
    panel.className = 'log-panel';
    
    const header = document.createElement('div');
    header.className = 'log-panel-header';
    
    const titleElement = document.createElement('h4');
    titleElement.textContent = title;
    
    const statusElement = document.createElement('span');
    statusElement.className = `log-status ${status}`;
    statusElement.textContent = status === 'active' ? '실행 중' : '대기 중';
    
    header.appendChild(titleElement);
    header.appendChild(statusElement);
    
    const content = document.createElement('div');
    content.className = 'log-panel-content';
    content.id = `grid-log-${orderId}`;
    
    panel.appendChild(header);
    panel.appendChild(content);
    
    return panel;
}

// 페이지 로드 시 그리드 로그 초기화
document.addEventListener('DOMContentLoaded', () => {
    initializeGridLogs();
});

// Logout function
function logout(): void {
    // 웹소켓 연결 종료
    stopWebSocketConnection();
    
    currentUser = null;
    localStorage.removeItem('userID');
    showPage('home');
    // 로그아웃 시 알림 제거
    const existingAlert = document.querySelector('.alert');
    if (existingAlert) {
        existingAlert.remove();
    }
}

// Check if user is logged in
function checkAuth(): void {
    const userID = localStorage.getItem('userID');
    if (!userID) {
        // Redirect to login if not authenticated
        showPage('login');
    }
}

// Initialize app
function initApp(): void {
    // Add fade-in animation to cards
    const cards = document.querySelectorAll('.welcome-card, .login-card, .register-card, .dashboard-card');
    cards.forEach(card => {
        card.classList.add('fade-in');
    });

    // Check if user is already logged in
    const userID = localStorage.getItem('userID');
    if (userID) {
        showPage('dashboard');
        getUserInfo(userID);
    }
}

// Event listeners
document.addEventListener('DOMContentLoaded', function() {
    initApp();
});

// Add smooth scrolling for anchor links
document.addEventListener('click', function(e) {
    const target = e.target as HTMLElement;
    const href = target.getAttribute('href');
    if (target.tagName === 'A' && href && href.startsWith('#') && href !== '#') {
        e.preventDefault();
        const targetElement = document.querySelector(href);
        if (targetElement) {
            targetElement.scrollIntoView({
                behavior: 'smooth'
            });
        }
    }
});

// Add form validation on input
document.addEventListener('input', function(e) {
    const target = e.target as HTMLElement;
    if (target.tagName === 'INPUT') {
        const form = target.closest('form');
        if (form) {
            const submitBtn = form.querySelector('button[type="submit"]') as HTMLButtonElement;
            if (submitBtn) {
                const formData = new FormData(form);
                const hasData = Array.from(formData.values()).some(value => value.toString().trim() !== '');
                submitBtn.disabled = !hasData;
            }
        }
    }
});

// Add keyboard shortcuts
document.addEventListener('keydown', function(e) {
    // Ctrl/Cmd + Enter to submit forms
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const form = document.querySelector('form');
        if (form) {
            form.dispatchEvent(new Event('submit'));
        }
    }
    
    // Escape to go back
    if (e.key === 'Escape') {
        const currentPage = document.querySelector('.page.active');
        if (currentPage && currentPage.id !== 'home-page') {
            showPage('home');
        }
    }
});

// Make functions globally available
(window as any).handleLogin = handleLogin;
(window as any).handleRegister = handleRegister;
(window as any).showPage = showPage;
(window as any).showDashboardTab = showDashboardTab;
(window as any).showAddPlatformModal = showAddPlatformModal;
(window as any).showEditPlatformModal = showEditPlatformModal;
(window as any).closePlatformModal = closePlatformModal;
(window as any).handlePlatformSubmit = handlePlatformSubmit;
(window as any).removePlatform = removePlatform;
(window as any).addSellOrder = addSellOrder;
(window as any).closeSellOrderModal = closeSellOrderModal;
(window as any).handleSellOrderSubmit = handleSellOrderSubmit;
(window as any).toggleOrderDetail = toggleOrderDetail;

(window as any).viewInModal = viewInModal;
(window as any).closeLogModal = closeLogModal;

(window as any).logout = logout;

// 현재 플랫폼 목록 저장용
let currentPlatforms: PlatformKey[] = [];

 