import * as path from 'path';
import * as vscode from 'vscode';
import { Subscription } from 'rxjs';
import { AzureDevApplication, AzureDevApplicationProvider } from '../../services/AzureDevApplicationProvider';
import { WorkspaceResource, WorkspaceResourceProvider } from './ResourceGroupsApi';

export class AzureDevCliWorkspaceResourceProvider extends vscode.Disposable implements WorkspaceResourceProvider {
    private readonly onDidChangeResourceEmitter = new vscode.EventEmitter<WorkspaceResource | undefined>();
    private readonly applicationsSubscription: Subscription;

    private applications: AzureDevApplication[] = [];

    constructor(applicationProvider: AzureDevApplicationProvider) {
        super(
            () => {
                this.applicationsSubscription.unsubscribe();
                this.onDidChangeResourceEmitter.dispose();
            });

        this.applicationsSubscription =
            applicationProvider
                .applications
                .subscribe(
                    applications => {
                        this.applications = applications;
                        this.onDidChangeResourceEmitter.fire(undefined);
                    });
    }

    readonly onDidChangeResource = this.onDidChangeResourceEmitter.event;

    async getResources(source: vscode.WorkspaceFolder): Promise<WorkspaceResource[]> {
        const resources: WorkspaceResource[] = [];

        for (const application of this.applications.filter(application => application.workspaceFolder === source)) {
            const configurationFilePath = application.configurationPath.fsPath;
            const configurationFolderName = path.basename(path.dirname(configurationFilePath));

            resources.push({
                folder: source,
                id: application.configurationPath.fsPath,
                name: configurationFolderName,
                resourceType: 'ms-azuretools.azure-dev.application'
            });
        }

        return resources;
    }
}