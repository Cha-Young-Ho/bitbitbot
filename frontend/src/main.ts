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
    type?: string;
    error?: string;
}

interface PlatformKey {
    platformName: string;
    name: string;
    platformAccessKey: string;
    platformSecretKey: string;
    passwordPhrase: string;
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

async function callAddPlatform(userID: string, platform: string, name: string, accessKey: string, secretKey: string, passwordPhrase: string): Promise<any> {
    try {
        const { AddPlatform } = await import('../wailsjs/go/main/App');
        const result = await AddPlatform(userID, platform, name, accessKey, secretKey, passwordPhrase);
        return result;
    } catch (error) {
        console.error('AddPlatform error:', error);
        return { success: false, message: '플랫폼 추가 중 오류가 발생했습니다.' };
    }
}

async function callUpdatePlatform(userID: string, oldPlatform: string, oldName: string, newPlatform: string, newName: string, accessKey: string, secretKey: string, passwordPhrase: string): Promise<any> {
    try {
        const { UpdatePlatform } = await import('../wailsjs/go/main/App');
        const result = await UpdatePlatform(userID, oldPlatform, oldName, newPlatform, newName, accessKey, secretKey, passwordPhrase);
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
    console.log(`[${type.toUpperCase()}] ${message}`);
    
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
            // Store user info (세션 기준)
            currentUser = result.user;
            sessionStorage.setItem('userID', formData.userID);
            
            showAlert(result.message, 'success');
            
            // Redirect to dashboard
            setTimeout(() => {
                showPage('dashboard');
                getUserInfo(formData.userID);
            }, 1000);
        } else {
            // 접근 권한 문제인 경우
            if (result && result.type === 'invalid_access') {
                showInvalidAccessDialog();
                return;
            }
            
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
                            <button class="btn btn-secondary" onclick="showEditPlatformModal('${platform.platformName}', '${platform.name}', '${platform.platformAccessKey}', '${platform.platformSecretKey}', '${platform.passwordPhrase || ''}')">
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
                        <div class="platform-detail">
                            <div class="platform-detail-label">Password Phrase</div>
                            <div class="platform-detail-value">${platform.passwordPhrase || '설정되지 않음'}</div>
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
            'file-management': '파일 관리',
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
                const userID = sessionStorage.getItem('userID');
                if (userID) {
                    loadSellOrders(userID);
                    // 웹소켓 연결 시작
                    startWebSocketConnection(userID);
                }
            }
            
            // 파일 관리 탭이 활성화되면 파일 데이터 로드
            if (tabName === 'file-management') {
                const userID = sessionStorage.getItem('userID');
                if (userID) {
                    loadFileData(userID);
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

function showEditPlatformModal(platformName: string, name: string, accessKey: string, secretKey: string, passwordPhrase: string): void {
    // HTML 엔티티 디코딩
    const decodedPlatformName = platformName.replace(/&apos;/g, "'");
    const decodedName = name.replace(/&apos;/g, "'");
    const decodedAccessKey = accessKey.replace(/&apos;/g, "'");
    const decodedSecretKey = secretKey.replace(/&apos;/g, "'");
    const decodedPasswordPhrase = passwordPhrase.replace(/&apos;/g, "'");
    
    currentEditingPlatform = { 
        platformName: decodedPlatformName, 
        name: decodedName, 
        platformAccessKey: decodedAccessKey, 
        platformSecretKey: decodedSecretKey,
        passwordPhrase: decodedPasswordPhrase
    };
    document.getElementById('modal-title')!.textContent = '플랫폼 수정';
    document.getElementById('platform-submit-btn')!.textContent = '수정';
    
    // 폼에 기존 값들 채우기
    const form = document.getElementById('platform-form') as HTMLFormElement;
    (form.querySelector('[name="platformName"]') as HTMLSelectElement).value = decodedPlatformName;
    (form.querySelector('[name="name"]') as HTMLInputElement).value = decodedName;
    (form.querySelector('[name="accessKey"]') as HTMLInputElement).value = decodedAccessKey;
    (form.querySelector('[name="secretKey"]') as HTMLInputElement).value = decodedSecretKey;
    (form.querySelector('[name="passwordPhrase"]') as HTMLInputElement).value = decodedPasswordPhrase;
    
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
    const passwordPhrase = formData.get('passwordPhrase') as string;
    
    // 입력값 검증
    if (!platformName || !name || !accessKey || !secretKey) {
        showAlert('모든 필드를 입력해주세요.', 'error');
        return;
    }
    
    // 로딩 표시
    showLoading(submitBtn);
    
    try {
        const userID = sessionStorage.getItem('userID');
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
                secretKey,
                passwordPhrase
            );
        } else {
            // 추가 모드
            result = await callAddPlatform(userID, platformName, name, accessKey, secretKey, passwordPhrase);
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
        const userID = sessionStorage.getItem('userID');
        
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

function openEditOrderModal(order: SellOrder): void {
    const modal = document.getElementById('sell-order-modal');
    if (!modal) return;
    // 폼 채우기
    const form = document.getElementById('sell-order-form') as HTMLFormElement;
    (form.querySelector('input[name="orderName"]') as HTMLInputElement).value = order.name;
    (form.querySelector('input[name="symbol"]') as HTMLInputElement).value = order.symbol;
    (form.querySelector('input[name="price"]') as HTMLInputElement).value = String(order.price);
    (form.querySelector('input[name="quantity"]') as HTMLInputElement).value = String(order.quantity);
    (form.querySelector('input[name="term"]') as HTMLInputElement).value = String(order.term);
    const select = document.getElementById('platformKey-select') as HTMLSelectElement;
    select.value = `${order.platform}-${order.platformNickName}`;
    (modal as HTMLElement).classList.add('active');
    (modal as any).dataset.editing = 'true';
    (modal as any).dataset.oldName = order.name;
}

function loadPlatformKeys(): void {
    const userID = sessionStorage.getItem('userID');
    
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
    // 심볼 검증: BASE/QUOTE 형식, QUOTE는 KRW|USDT|BTC
    const symbolOk = /^[A-Za-z0-9]+\/(KRW|USDT|BTC)$/.test(symbol || '');
    if (!platformKey || !orderName || !symbol || !symbolOk || isNaN(price) || isNaN(quantity) || isNaN(term)) {
        console.log('입력값 검증 실패:', { platformKey, orderName, symbol, price, quantity, term });
        showAlert('심볼 형식은 BASE/QUOTE 이며, QUOTE는 KRW, USDT, BTC만 허용됩니다.', 'error');
        return;
    }
    console.log('입력값 검증 통과');
    
    // 로딩 표시
    showLoading(submitBtn);
    
    try {
        const userID = sessionStorage.getItem('userID');
        if (!userID) {
            showAlert('로그인이 필요합니다.', 'error');
            return;
        }
        
        // 플랫폼 정보 파싱
        const [platformName, platformNickName] = platformKey.split('-');
        
        // 편집 모드 여부
        const modalEl = document.getElementById('sell-order-modal') as any;
        const isEditing = modalEl?.dataset?.editing === 'true';
        let result: any;
        if (isEditing) {
            const oldName = modalEl.dataset.oldName as string;
            result = await callUpdateSellOrder(userID, oldName, orderName, symbol, price, quantity, term, platformName, platformNickName);
            delete modalEl.dataset.editing;
            delete modalEl.dataset.oldName;
        } else {
            // 예약 매도 주문 추가
            result = await callAddSellOrder(userID, orderName, symbol, price, quantity, term, platformName, platformNickName);
        }
        
        if (result.success) {
            showAlert(result.message || (isEditing ? '수정되었습니다.' : '추가되었습니다.'), 'success');
            closeSellOrderModal();
            // 예약 매도 목록 새로고침
            await loadSellOrders(userID);
            renderSellOrders();
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

async function callUpdateSellOrder(userID: string, oldName: string, orderName: string, symbol: string, price: number, quantity: number, term: number, platformName: string, platformNickName: string): Promise<any> {
    try {
        const { UpdateSellOrder } = await import('../wailsjs/go/main/App');
        const result = await UpdateSellOrder(userID, oldName, orderName, symbol, price, quantity, term, platformName, platformNickName);
        return result;
    } catch (error) {
        console.error('UpdateSellOrder error:', error);
        return { success: false, message: '예약 매도 수정 중 오류가 발생했습니다.' };
    }
}

async function callRemoveSellOrder(userID: string, orderName: string): Promise<any> {
    try {
        const { RemoveSellOrder } = await import('../wailsjs/go/main/App');
        const result = await RemoveSellOrder(userID, orderName);
        return result;
    } catch (error) {
        console.error('RemoveSellOrder error:', error);
        return { success: false, message: '예약 매도 삭제 중 오류가 발생했습니다.' };
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

// (삭제) 워커 로그 스트리밍 API 호출 제거: 웹소켓 통합으로 불필요

// 특정 주문의 로그 조회 API 호출
// (삭제) 주문 로그 조회 API 호출 제거: 웹소켓 통합으로 불필요

// (삭제) 구독/통합 로그 API 호출 제거: 웹소켓 통합으로 불필요

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
    const userID = sessionStorage.getItem('userID');
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
    // (삭제) 구 API 호출 제거: 웹소켓에서 실시간 로그 처리
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
                const payload = JSON.parse(event.data);
                console.log('웹소켓 수신:', payload);

                // 단일 통합 포맷 분기 처리
                if (payload && payload.category) {
                    if (payload.category === 'orderLog') {
                        handleOrderOutbound(payload);
                    } else if (payload.category === 'systemLog') {
                        handleSystemOutbound(payload);
                    } else {
                        console.warn('알 수 없는 category:', payload.category);
                    }
                } else {
                    // 구 포맷 호환 처리
                    processUnifiedLog(payload);
                }
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
    // 디버그 로그 제거
    // OrderName으로 해당 주문 찾기
    const order = sellOrders.find(o => o.id === log.orderName);
    
    // OrderName으로 찾지 못한 경우 플랫폼과 별칭으로 찾기 (백업)
    const orderByPlatform = !order ? sellOrders.find(o => 
        o.platform.toLowerCase() === log.platform.toLowerCase() && 
        o.platformNickName === log.nickname
    ) : null;
    
    const targetOrder = order || orderByPlatform;
    
    if (targetOrder) {
        // 해당 주문의 로그 컨테이너에 로그 추가 (예약 매도 관리)
        addLogToOrder(targetOrder.id, log);
        
        // 그리드 로그 뷰어에도 로그 추가
        addLogToGrid(targetOrder.id, log);
    } else {
        console.log('해당하는 주문을 찾을 수 없음:', log.orderName, log.platform, log.nickname);
    }
}

// 신규 포맷: 주문 로그 처리
function handleOrderOutbound(envelope: { category: string; data: any }): void {
    const data = envelope?.data ?? {};
    // data는 백엔드 WorkerLog 포맷
    // { orderName, platform, message, logType, timestamp, ... }
    const order = sellOrders.find(o => o.id === data.orderName || o.name === data.orderName);
    if (!order) {
        console.log('해당 주문을 찾지 못했습니다(신규포맷):', data.orderName, data.platform);
        return;
    }
    const normalized = {
        timestamp: new Date(data.timestamp),
        message: data.message ?? '',
        logType: data.logType ?? 'info'
    };
    addLogToOrder(order.id, normalized);
    addLogToGrid(order.id, normalized);
}

// 신규 포맷: 시스템 로그 처리
function handleSystemOutbound(envelope: { category: string; data: any }): void {
    const data = envelope?.data ?? {};
    const normalized = {
        timestamp: new Date(data.timestamp),
        logType: data.logType ?? 'info',
        message: data.message ?? ''
    };
    addSystemLog(normalized);
}

// 주문에 로그 추가 (예약 매도 관리)
function addLogToOrder(orderName: string, log: any): void {
    const logContainer = document.getElementById(`${orderName}-logs`);
    if (!logContainer) return;
    
    // 스크롤 추적 설정 (처음 한 번만)
    const containerId = `${orderName}-logs`;
    if (!scrollStates.has(containerId)) {
        setupScrollTracking(containerId);
    }
    
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${log.logType || 'info'}`;
    
    const timestamp = new Date(log.timestamp).toLocaleTimeString();
    const message = log.message;
    
    // 통일된 로그 포맷: [시간] 메시지
    logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
    logContainer.appendChild(logEntry);
    
    // 자동 스크롤 수행
    autoScrollToBottom(containerId);
    
    // 최대 300개 로그만 유지
    const logEntries = logContainer.querySelectorAll('.log-entry');
    if (logEntries.length > 300) {
        logEntries[0].remove();
    }
}

// 그리드에 로그 추가 (로그 뷰어)
function addLogToGrid(orderName: string, log: any): void {
    const gridLogContainer = document.getElementById(`${orderName}-grid-logs`);
    if (!gridLogContainer) {
        console.log('그리드 로그 컨테이너를 찾을 수 없음:', `${orderName}-grid-logs`);
        return;
    }
    
    // 스크롤 추적 설정 (처음 한 번만)
    const containerId = `${orderName}-grid-logs`;
    if (!scrollStates.has(containerId)) {
        setupScrollTracking(containerId);
    }
    
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry ${log.logType || 'info'}`;
    
    const timestamp = new Date(log.timestamp).toLocaleTimeString();
    const message = log.message;
    
    // 통일된 로그 포맷: [시간] 메시지
    logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
    gridLogContainer.appendChild(logEntry);
    
    // 자동 스크롤 수행
    autoScrollToBottom(containerId);
    
    // 최대 50개 로그만 유지
    const logEntries = gridLogContainer.querySelectorAll('.log-entry');
    if (logEntries.length > 50) {
        logEntries[0].remove();
    }
}

// 시스템 로그 추가 (신규 및 구 포맷 공통 사용)
function addSystemLog(log: { timestamp: Date | string | number; message: string; logType?: string }): void {
    const systemLogContainer = document.getElementById('system-log-container');
    if (!systemLogContainer) return;
    
    // 스크롤 추적 설정 (처음 한 번만)
    const containerId = 'system-log-container';
    if (!scrollStates.has(containerId)) {
        setupScrollTracking(containerId);
    }
    
    const logEntry = document.createElement('div');
    logEntry.className = `system-log-entry ${log.logType || 'info'}`;
    const ts = new Date(log.timestamp).toLocaleTimeString();
    // 통일된 로그 포맷: [시간] 메시지
    logEntry.innerHTML = `<span class="timestamp">[${ts}]</span> ${log.message}`;
    systemLogContainer.appendChild(logEntry);
    
    // 자동 스크롤 수행
    autoScrollToBottom(containerId);
    
    const logEntries = systemLogContainer.querySelectorAll('.system-log-entry');
    if (logEntries.length > 100) logEntries[0].remove();
}

// (삭제) 주문 로그 조회/표시 관련 구 API 로직 제거: 웹소켓 통합으로 불필요

// 모달 로그 표시
function displayModalLogs(logs: any[]): void {
    const logContent = document.getElementById('log-content');
    if (!logContent) return;
    
    // 스크롤 추적 설정 (처음 한 번만)
    const containerId = 'log-content';
    if (!scrollStates.has(containerId)) {
        setupScrollTracking(containerId);
    }
    
    // 기존 로그 제거
    logContent.innerHTML = '';
    
    // 최근 20개 로그 표시
    const recentLogs = logs.slice(-20);
    
    recentLogs.forEach(log => {
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry ${log.logType || 'info'}`;
        
        const timestamp = new Date(log.timestamp).toLocaleTimeString();
        const message = log.message;
        
        // 통일된 로그 포맷: [시간] 메시지
        logEntry.innerHTML = `<span class="timestamp">[${timestamp}]</span> ${message}`;
        logContent.appendChild(logEntry);
    });
    
    // 자동 스크롤 수행
    autoScrollToBottom(containerId);
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
    // 수정 버튼
    const editBtn = document.createElement('button');
    editBtn.className = 'btn btn-secondary';
    editBtn.textContent = '수정';
    editBtn.onclick = (e) => openEditOrderModal(order);
    // 삭제 버튼
    const delBtn = document.createElement('button');
    delBtn.className = 'btn btn-danger';
    delBtn.textContent = '삭제';
    delBtn.onclick = async (e) => {
        e.stopPropagation();
        const userID = sessionStorage.getItem('userID');
        if (!userID) { showAlert('로그인이 필요합니다.', 'error'); return; }
        const res = await callRemoveSellOrder(userID, order.name);
        if (res.success) {
            showAlert('삭제되었습니다.', 'success');
            await loadSellOrders(userID);
            renderSellOrders();
        } else {
            showAlert(res.message || '삭제 실패', 'error');
        }
    };
    orderActions.appendChild(editBtn);
    orderActions.appendChild(delBtn);
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
    const userID = sessionStorage.getItem('userID');
    if (userID) {
    // (삭제) 구 API 호출 제거: 웹소켓에서 실시간 로그 처리
    }
    console.log();
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
            const userID = sessionStorage.getItem('userID');
            if (userID) {
    // (삭제) 구 API 호출 제거: 웹소켓에서 실시간 로그 처리
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
    console.log('로그 뷰어 그리드 초기화 시작');
    const container = document.getElementById('log-grid-container');
    if (!container) {
        console.log('로그 그리드 컨테이너를 찾을 수 없음');
        return;
    }
    
    container.innerHTML = '';
    
    console.log('현재 주문 목록:', sellOrders);
    
    // 각 주문에 대한 로그 패널 생성
    sellOrders.forEach((order, index) => {
        const panel = createLogPanel(order.id, `${order.name} (${order.platform})`, order.status);
        container.appendChild(panel);
        console.log('로그 패널 생성됨:', order.id);
    });
    
    console.log('로그 뷰어 그리드 초기화 완료');
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
    content.id = `${orderId}-grid-logs`;
    
    panel.appendChild(header);
    panel.appendChild(content);
    
    return panel;
}

// 페이지 로드 시 그리드 로그 초기화
document.addEventListener('DOMContentLoaded', () => {
    initializeGridLogs();
});

// Logout function
async function logout(): Promise<void> {
    const userID = sessionStorage.getItem('userID');
    if (userID) {
        try {
            const { Logout } = await import('../wailsjs/go/main/App');
            await Logout(userID);
        } catch (e) {
            console.error('Logout API error:', e);
        }
    }
    // 웹소켓 연결 종료
    stopWebSocketConnection();
    currentUser = null;
    sessionStorage.removeItem('userID');
    showPage('home');
    // 로그아웃 시 알림 제거
    const existingAlert = document.querySelector('.alert');
    if (existingAlert) {
        existingAlert.remove();
    }
}

// Check if user is logged in
function checkAuth(): void {
    const userID = sessionStorage.getItem('userID');
    if (!userID) {
        // Redirect to login if not authenticated
        showPage('login');
    }
}

// 시스템 로그 지우기
function clearSystemLog(): void {
    const systemLogContainer = document.getElementById('system-log-container');
    if (systemLogContainer) {
        systemLogContainer.innerHTML = '';
    }
}

// 모든 그리드 로그 지우기
function clearAllGridLogs(): void {
    sellOrders.forEach(order => {
        const gridLogContainer = document.getElementById(`${order.id}-grid-logs`);
        if (gridLogContainer) {
            gridLogContainer.innerHTML = '';
        }
    });
}

// Initialize app
function initApp(): void {
    // Add fade-in animation to cards
    const cards = document.querySelectorAll('.welcome-card, .login-card, .register-card, .dashboard-card');
    cards.forEach(card => {
        card.classList.add('fade-in');
    });

    // 강제 비로그인 시작: 자동로그인 해제
    sessionStorage.removeItem('userID');
    showPage('login');
}

// Event listeners
document.addEventListener('DOMContentLoaded', async function() {
    // 업데이트 체크
    await checkForUpdatesOnStartup();
    
    // 주기적 알림 확인 (5초마다)
    setInterval(checkPeriodicNotifications, 5000);
    
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

// 파일 관리 관련 함수들
async function callGetLocalFileData(userID: string): Promise<any> {
    try {
        const { GetLocalFileData } = await import('../wailsjs/go/main/App');
        const result = await GetLocalFileData(userID);
        return result;
    } catch (error) {
        console.error('GetLocalFileData error:', error);
        return { success: false, message: '파일 데이터를 불러오는 중 오류가 발생했습니다.' };
    }
}

async function callSaveLocalFileData(userID: string, jsonData: string): Promise<any> {
    try {
        const { SaveLocalFileData } = await import('../wailsjs/go/main/App');
        const result = await SaveLocalFileData(userID, jsonData);
        return result;
    } catch (error) {
        console.error('SaveLocalFileData error:', error);
        return { success: false, message: '파일 저장 중 오류가 발생했습니다.' };
    }
}

// 파일 데이터 로드
async function loadFileData(userID: string): Promise<void> {
    try {
        const result = await callGetLocalFileData(userID);
        if (result.success) {
            const textarea = document.getElementById('json-editor') as HTMLTextAreaElement;
            if (textarea) {
                textarea.value = result.data;
                validateJSON();
                // 실시간 유효성 검사 이벤트 리스너 추가
                textarea.addEventListener('input', validateJSON);
            }
        } else {
            showToast(result.message, 'error');
        }
    } catch (error) {
        console.error('파일 데이터 로드 오류:', error);
        showToast('파일 데이터를 불러오는 중 오류가 발생했습니다.', 'error');
    }
}

// JSON 유효성 검사
function validateJSON(): void {
    const textarea = document.getElementById('json-editor') as HTMLTextAreaElement;
    const statusElement = document.getElementById('json-status-text');
    const statusContainer = document.querySelector('.json-status');
    
    if (!textarea || !statusElement || !statusContainer) return;
    
    try {
        const jsonText = textarea.value.trim();
        if (jsonText === '') {
            statusElement.textContent = 'JSON 데이터를 입력하세요';
            statusContainer.className = 'json-status warning';
            textarea.className = '';
            return;
        }
        
        JSON.parse(jsonText);
        statusElement.textContent = 'JSON 형식이 올바릅니다';
        statusContainer.className = 'json-status valid';
        textarea.className = 'valid';
    } catch (error) {
        statusElement.textContent = `JSON 형식 오류: ${error instanceof Error ? error.message : '알 수 없는 오류'}`;
        statusContainer.className = 'json-status error';
        textarea.className = 'error';
    }
}

// JSON 포맷 정리
function formatJSON(): void {
    const textarea = document.getElementById('json-editor') as HTMLTextAreaElement;
    if (!textarea) return;
    
    try {
        const jsonText = textarea.value.trim();
        if (jsonText === '') {
            showToast('JSON 데이터가 없습니다.', 'warning');
            return;
        }
        
        const parsed = JSON.parse(jsonText);
        const formatted = JSON.stringify(parsed, null, 2);
        textarea.value = formatted;
        validateJSON();
        showToast('JSON 포맷이 정리되었습니다.', 'success');
    } catch (error) {
        showToast('잘못된 JSON 형식입니다.', 'error');
        validateJSON();
    }
}

// 파일 데이터 저장
async function saveFileData(): Promise<void> {
    const userID = sessionStorage.getItem('userID');
    if (!userID) {
        showToast('로그인이 필요합니다.', 'error');
        return;
    }
    
    const textarea = document.getElementById('json-editor') as HTMLTextAreaElement;
    if (!textarea) return;
    
    try {
        const jsonData = textarea.value.trim();
        if (jsonData === '') {
            showToast('저장할 JSON 데이터가 없습니다.', 'warning');
            return;
        }
        
        // JSON 유효성 검사
        JSON.parse(jsonData);
        
        const result = await callSaveLocalFileData(userID, jsonData);
        if (result.success) {
            showToast('파일이 성공적으로 저장되었습니다.', 'success');
            // 플랫폼과 주문 데이터 새로고침
            const userInfo = await callGetUserInfo(userID);
            if (userInfo.success && userInfo.user) {
                displayUserInfo(userInfo.user);
            }
            await loadSellOrders(userID);
        } else {
            showToast(result.message, 'error');
        }
    } catch (error) {
        showToast('잘못된 JSON 형식입니다.', 'error');
        validateJSON();
    }
}

// 토스트 메시지 표시
function showToast(message: string, type: 'success' | 'error' | 'warning' = 'success'): void {
    // 기존 토스트 제거
    const existingToast = document.querySelector('.toast');
    if (existingToast) {
        existingToast.remove();
    }
    
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    
    document.body.appendChild(toast);
    
    // 3초 후 자동 제거
    setTimeout(() => {
        if (toast.parentNode) {
            toast.remove();
        }
    }, 3000);
}

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
(window as any).formatJSON = formatJSON;
(window as any).saveFileData = saveFileData;

(window as any).viewInModal = viewInModal;
(window as any).closeLogModal = closeLogModal;

(window as any).logout = logout;

// 현재 플랫폼 목록 저장용
let currentPlatforms: PlatformKey[] = [];

// 스크롤 상태 추적을 위한 변수들
const scrollStates = new Map<string, { userScrolled: boolean; lastScrollTop: number }>();

// 스크롤 이벤트 리스너를 추가하는 함수
function setupScrollTracking(containerId: string): void {
    const container = document.getElementById(containerId);
    if (!container) return;
    
    // 초기 상태 설정
    scrollStates.set(containerId, { userScrolled: false, lastScrollTop: 0 });
    
    // 스크롤 이벤트 리스너 추가
    container.addEventListener('scroll', () => {
        const state = scrollStates.get(containerId);
        if (!state) return;
        
        const currentScrollTop = container.scrollTop;
        const isAtBottom = currentScrollTop + container.clientHeight >= container.scrollHeight - 10;
        
        // 사용자가 스크롤을 움직였는지 확인 (더 정확한 감지)
        const scrollDifference = Math.abs(currentScrollTop - state.lastScrollTop);
        if (scrollDifference > 10) { // 10px 이상 움직였을 때만 사용자 스크롤로 인식
            // 맨 아래에 있으면 자동 스크롤 활성화, 그렇지 않으면 비활성화
            state.userScrolled = !isAtBottom;
        }
        
        state.lastScrollTop = currentScrollTop;
    });
    
    // 마우스 휠 이벤트도 감지 (더 정확한 사용자 의도 파악)
    container.addEventListener('wheel', () => {
        const state = scrollStates.get(containerId);
        if (!state) return;
        
        const currentScrollTop = container.scrollTop;
        const isAtBottom = currentScrollTop + container.clientHeight >= container.scrollHeight - 10;
        
        // 휠 이벤트가 발생하면 사용자가 스크롤을 의도했다고 판단
        state.userScrolled = !isAtBottom;
        state.lastScrollTop = currentScrollTop;
    });
}

// 자동 스크롤을 수행하는 함수
function autoScrollToBottom(containerId: string): void {
    const container = document.getElementById(containerId);
    if (!container) return;
    
    const state = scrollStates.get(containerId);
    if (!state) return;
    
    // 사용자가 스크롤을 움직이지 않았거나 맨 아래에 있을 때만 자동 스크롤
    if (!state.userScrolled) {
        // 부드러운 스크롤을 위해 requestAnimationFrame 사용
        requestAnimationFrame(() => {
            container.scrollTop = container.scrollHeight;
            // 스크롤 후 상태 업데이트
            state.lastScrollTop = container.scrollTop;
        });
    }
}

// 업데이트 다이얼로그 관련 함수들
function showUpdateDialog(isRequired: boolean, currentVersion: string, requiredVersion: string): Promise<boolean> {
    return new Promise((resolve) => {
        // 기존 다이얼로그가 있으면 제거
        const existingDialog = document.getElementById('update-dialog');
        if (existingDialog) {
            existingDialog.remove();
        }

        const dialog = document.createElement('div');
        dialog.id = 'update-dialog';
        dialog.className = 'update-dialog-overlay';
        
        const title = isRequired ? '필수 업데이트가 있습니다' : '업데이트가 있습니다';
        const message = isRequired 
            ? `업데이트를 하지 않으면 프로그램이 종료됩니다.\n현재 버전: ${currentVersion}\n필수 버전: ${requiredVersion}`
            : `업데이트하시겠습니까?\n현재 버전: ${currentVersion}\n권장 버전: ${requiredVersion}`;

        dialog.innerHTML = `
            <div class="update-dialog">
                <div class="update-dialog-header">
                    <h3>${title}</h3>
                </div>
                <div class="update-dialog-body">
                    <p>${message}</p>
                </div>
                <div class="update-dialog-footer">
                    <button id="update-btn" class="btn btn-primary">업데이트</button>
                    <button id="cancel-btn" class="btn btn-secondary">취소</button>
                </div>
            </div>
        `;

        document.body.appendChild(dialog);

        // 이벤트 리스너 추가
        document.getElementById('update-btn')?.addEventListener('click', () => {
            dialog.remove();
            resolve(true);
        });

        document.getElementById('cancel-btn')?.addEventListener('click', () => {
            dialog.remove();
            resolve(false);
        });
    });
}

// 업데이트 진행 상태 표시
function showUpdateProgress(): void {
    const existingProgress = document.getElementById('update-progress');
    if (existingProgress) {
        existingProgress.remove();
    }

    const progress = document.createElement('div');
    progress.id = 'update-progress';
    progress.className = 'update-progress-overlay';
    
    progress.innerHTML = `
        <div class="update-progress">
            <div class="update-progress-header">
                <h3>업데이트 중...</h3>
            </div>
            <div class="update-progress-body">
                <div class="progress-bar">
                    <div class="progress-fill"></div>
                </div>
                <p>새 버전을 다운로드하고 있습니다...</p>
            </div>
        </div>
    `;

    document.body.appendChild(progress);
}

// 업데이트 진행 상태 숨기기
function hideUpdateProgress(): void {
    const progress = document.getElementById('update-progress');
    if (progress) {
        progress.remove();
    }
}

// 재시작 안내 다이얼로그 표시
function showRestartDialog(): void {
    const existingDialog = document.getElementById('restart-dialog');
    if (existingDialog) {
        existingDialog.remove();
    }

    const dialog = document.createElement('div');
    dialog.id = 'restart-dialog';
    dialog.className = 'update-dialog-overlay';
    
    dialog.innerHTML = `
        <div class="update-dialog">
            <div class="update-dialog-header">
                <h3>업데이트 완료</h3>
            </div>
            <div class="update-dialog-body">
                <p>업데이트가 완료되었습니다.</p>
                <p>새로운 버전을 적용하려면 프로그램을 다시 시작해주세요.</p>
            </div>
            <div class="update-dialog-footer">
                <button id="confirm-btn" class="btn btn-primary">종료</button>
            </div>
        </div>
    `;

    document.body.appendChild(dialog);

    // 이벤트 리스너 추가
    document.getElementById('confirm-btn')?.addEventListener('click', () => {
        dialog.remove();
        // 프로그램 종료
        if ((window as any).runtime) {
            (window as any).runtime.Quit();
        } else {
            // fallback: window.close() 시도
            window.close();
        }
    });
}

// 잘못된 접근 다이얼로그 표시
function showInvalidAccessDialog(): void {
    const existingDialog = document.getElementById('invalid-access-dialog');
    if (existingDialog) {
        existingDialog.remove();
    }

    const dialog = document.createElement('div');
    dialog.id = 'invalid-access-dialog';
    dialog.className = 'update-dialog-overlay';
    
    dialog.innerHTML = `
        <div class="update-dialog">
            <div class="update-dialog-header">
                <h3>잘못된 접근</h3>
            </div>
            <div class="update-dialog-body">
                <p>접근 권한이 없거나 프로그램이 비활성화되었습니다.</p>
                <p>모든 워커가 중지되었습니다.</p>
            </div>
            <div class="update-dialog-footer">
                <button id="exit-btn" class="btn btn-primary">종료</button>
            </div>
        </div>
    `;

    document.body.appendChild(dialog);

    // 이벤트 리스너 추가
    document.getElementById('exit-btn')?.addEventListener('click', () => {
        dialog.remove();
        // 프로그램 종료
        if ((window as any).runtime) {
            (window as any).runtime.Quit();
        } else {
            // fallback: window.close() 시도
            window.close();
        }
    });
}

// 앱 시작 시 업데이트 체크
async function checkForUpdatesOnStartup(): Promise<void> {
    try {
        const result = await window.go.main.App.CheckForUpdates();
        
        if (!result.success) {
            // 접근 권한 문제인 경우
            if (result.type === 'invalid_access') {
                showInvalidAccessDialog();
                return;
            }
            // 기타 오류
            showToast(result.message, 'error');
            return;
        }
        
        if (result.updateRequired) {
            // 필수 업데이트인지 선택적 업데이트인지에 따라 다른 버전 정보 사용
            const versionToShow = result.isRequired ? result.requiredVersion : result.recommendedVersion;
            
            const shouldUpdate = await showUpdateDialog(
                result.isRequired,
                result.currentVersion,
                versionToShow
            );
            
            if (shouldUpdate) {
                await performUpdate();
            } else if (result.isRequired) {
                // 필수 업데이트에서 취소를 누른 경우 프로그램 종료
                if ((window as any).runtime) {
                    (window as any).runtime.Quit();
                } else {
                    window.close();
                }
            }
        }
    } catch (error) {
        console.error('업데이트 체크 실패:', error);
    }
}

// 주기적 알림 확인 및 처리
async function checkPeriodicNotifications(): Promise<void> {
    try {
        const result = await window.go.main.App.GetPeriodicValidationNotification();
        
        if (result.hasNotification) {
            if (result.type === 'required_update') {
                // 필수 업데이트 - 워커 중지 후 업데이트 다이얼로그
                await handleRequiredUpdate();
            } else if (result.type === 'optional_update') {
                // 선택적 업데이트 - 워커 유지하고 업데이트 다이얼로그
                await handleOptionalUpdate();
            } else if (result.type === 'invalid_access') {
                // 잘못된 접근 - 워커 중지 후 종료 다이얼로그
                showInvalidAccessDialog();
            }
        }
    } catch (error) {
        console.error('주기적 알림 확인 실패:', error);
    }
}

// 필수 업데이트 처리 (워커 중지 후 업데이트)
async function handleRequiredUpdate(): Promise<void> {
    try {
        const result = await window.go.main.App.CheckForUpdates();
        if (result.success && result.updateRequired) {
            const versionToShow = result.isRequired ? result.requiredVersion : result.recommendedVersion;
            const shouldUpdate = await showUpdateDialog(
                result.isRequired,
                result.currentVersion,
                versionToShow
            );
            
            if (shouldUpdate) {
                await performUpdate();
            } else {
                // 필수 업데이트에서 취소를 누른 경우 프로그램 종료
                if ((window as any).runtime) {
                    (window as any).runtime.Quit();
                } else {
                    window.close();
                }
            }
        }
    } catch (error) {
        console.error('필수 업데이트 처리 실패:', error);
    }
}

// 선택적 업데이트 처리 (워커 유지하고 업데이트)
async function handleOptionalUpdate(): Promise<void> {
    try {
        const result = await window.go.main.App.CheckForUpdates();
        if (result.success && result.updateRequired) {
            const versionToShow = result.isRequired ? result.requiredVersion : result.recommendedVersion;
            const shouldUpdate = await showUpdateDialog(
                result.isRequired,
                result.currentVersion,
                versionToShow
            );
            
            if (shouldUpdate) {
                await performUpdate();
            }
            // 선택적 업데이트에서 취소를 누른 경우 계속 사용
        }
    } catch (error) {
        console.error('선택적 업데이트 처리 실패:', error);
    }
}

// 주기적 검증 결과 처리
async function handlePeriodicValidation(): Promise<void> {
    try {
        const result = await window.go.main.App.CheckPeriodicValidation();
        
        if (!result.success) {
            if (result.type === 'version_update') {
                // 버전 업데이트 필요
                const updateResult = await window.go.main.App.CheckForUpdates();
                if (updateResult.success && updateResult.updateRequired) {
                    const versionToShow = updateResult.isRequired ? updateResult.requiredVersion : updateResult.recommendedVersion;
                    const shouldUpdate = await showUpdateDialog(
                        updateResult.isRequired,
                        updateResult.currentVersion,
                        versionToShow
                    );
                    
                    if (shouldUpdate) {
                        await performUpdate();
                    } else if (updateResult.isRequired) {
                        // 필수 업데이트에서 취소를 누른 경우 프로그램 종료
                        if ((window as any).runtime) {
                            (window as any).runtime.Quit();
                        } else {
                            window.close();
                        }
                    }
                }
            } else if (result.type === 'invalid_access') {
                // 잘못된 접근 - 워커 중지 및 종료 다이얼로그 표시
                showInvalidAccessDialog();
            } else {
                // 기타 오류
                showToast(result.message, 'error');
            }
        }
    } catch (error) {
        console.error('주기적 검증 처리 실패:', error);
    }
}

// 업데이트 수행
async function performUpdate(): Promise<void> {
    try {
        showUpdateProgress();
        
        const result = await window.go.main.App.PerformUpdate();
        
        if (result.success) {
            hideUpdateProgress();
            if (result.restartRequired) {
                showRestartDialog();
            } else {
                showToast('업데이트가 완료되었습니다.', 'success');
            }
        } else {
            hideUpdateProgress();
            showToast(result.message, 'error');
        }
    } catch (error) {
        hideUpdateProgress();
        console.error('업데이트 실패:', error);
        showToast('업데이트 중 오류가 발생했습니다.', 'error');
    }
}

 