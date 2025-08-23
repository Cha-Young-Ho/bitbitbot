// Wails 타입 정의
declare global {
    interface Window {
        go: {
            main: {
                App: {
                    Login: (userID: string, password: string) => Promise<any>;
                    Register: (userID: string, password: string) => Promise<any>;
                    GetUserInfo: (userID: string) => Promise<any>;
                    CheckForUpdates: () => Promise<any>;
                    PerformUpdate: () => Promise<any>;
                    CheckPeriodicValidation: () => Promise<any>;
                    GetPeriodicValidationNotification: () => Promise<any>;
                    GetFileLoadStatus: () => Promise<any>;
                    TestS3ConnectionFailure: () => Promise<any>;
                    GetS3FailureCountForTesting: () => Promise<any>;
                    ManualPeriodicCheck: () => Promise<any>;
                    StopAllWorkersAndExit: () => Promise<any>;
                };
            };
        };
    }
}

export {}; 