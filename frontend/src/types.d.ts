// Wails 타입 정의
declare global {
    interface Window {
        go: {
            main: {
                App: {
                    Login: (userID: string, password: string) => Promise<any>;
                    Register: (userID: string, password: string) => Promise<any>;
                    GetUserInfo: (userID: string) => Promise<any>;
                };
            };
        };
    }
}

export {}; 