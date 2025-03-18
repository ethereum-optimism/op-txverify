/**
 * Creates a modern transaction verification view from first principles
 * @param {Object} result - The verification result data
 * @returns {HTMLElement} The complete transaction verification UI
 */
function createVerificationResult(result) {
    // Format the data
    const formattedData = formatResultForDisplay(result);
    
    // Create container
    const container = document.createElement('div');
    container.style.cssText = `
        font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        background: #ffffff;
        color: #1f2937;
        border-radius: 16px;
        box-shadow: 0 10px 25px rgba(0,0,0,0.1);
        max-width: 960px;
        margin: 0 auto;
        overflow: hidden;
        --primary-color: ${formattedData.hasNestedTransaction ? '#ef4444' : '#10b981'};
        --bg-subtle: ${formattedData.hasNestedTransaction ? '#fff5f5' : '#f0fdf4'};
        --bg-strong: ${formattedData.hasNestedTransaction ? '#fee2e2' : '#dcfce7'};
    `;
    
    // Status banner
    container.appendChild(createStatusBanner(formattedData));
    
    // Transaction summary
    container.appendChild(createTransactionSummary(formattedData));
    
    // Main content area
    const contentArea = document.createElement('div');
    contentArea.style.cssText = `
        padding: 24px;
        display: flex;
        flex-direction: column;
        gap: 24px;
    `;
    
    // Transaction details
    contentArea.appendChild(createDetailsSection(formattedData));
    
    // Function call information
    if (formattedData.callDetails) {
        contentArea.appendChild(createFunctionCallSection(formattedData.callDetails));
    }
    
    // Nested transaction warning (if present)
    if (formattedData.hasNestedTransaction) {
        contentArea.appendChild(createNestedTransactionSection(formattedData.nestedInfo));
    }
    
    // Security verification section
    contentArea.appendChild(createSecuritySection(formattedData.hashes));
    
    container.appendChild(contentArea);
    return container;
}

/**
 * Creates the status banner showing risk level
 * @param {Object} data - The formatted data
 * @returns {HTMLElement} Status banner element
 */
function createStatusBanner(data) {
    const banner = document.createElement('div');
    banner.style.cssText = `
        background: var(--bg-strong);
        padding: 16px 24px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        border-bottom: 1px solid rgba(0,0,0,0.05);
    `;
    
    // Status information
    const statusInfo = document.createElement('div');
    statusInfo.style.cssText = `
        display: flex;
        align-items: center;
        gap: 12px;
    `;
    
    // Status icon
    const statusIcon = document.createElement('div');
    statusIcon.innerHTML = data.hasNestedTransaction ? 
        '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 9V12M12 16H12.01M8.21 13.89L7 23L12 20L17 23L15.79 13.88M11.5 3.08C11.82 3.03 12.18 3.03 12.5 3.08C15.1 3.4 17 5.62 17 8.28V9.61C17 9.8 17 9.9 17.04 9.98C17.07 10.06 17.11 10.12 17.21 10.24C17.3 10.36 17.42 10.47 17.67 10.67L18.93 11.69C19.45 12.11 19.5 12.88 19.05 13.37L18.04 14.47C17.82 14.72 17.47 14.88 17.09 14.89C16.7 14.9 16.31 14.75 16.06 14.5C16 14.44 15.95 14.39 15.92 14.35C15.89 14.32 15.88 14.31 15.87 14.29C15.86 14.28 15.85 14.27 15.84 14.24C15.78 14.13 15.77 14.02 15.77 13.89V8.28C15.77 6.38 14.58 4.75 12.93 4.53C12.65 4.49 12.35 4.49 12.07 4.53C10.42 4.75 9.23 6.38 9.23 8.28V13.89C9.23 14.02 9.22 14.13 9.16 14.24C9.15 14.27 9.14 14.28 9.13 14.29C9.12 14.31 9.11 14.32 9.08 14.35C9.05 14.39 9 14.44 8.94 14.5C8.69 14.75 8.3 14.9 7.91 14.89C7.53 14.88 7.18 14.72 6.96 14.47L5.95 13.37C5.5 12.88 5.55 12.11 6.07 11.69L7.33 10.67C7.58 10.47 7.7 10.36 7.79 10.24C7.89 10.12 7.93 10.06 7.96 9.98C8 9.9 8 9.8 8 9.61V8.28C8 5.62 9.9 3.4 12.5 3.08C12.18 3.03 11.82 3.03 11.5 3.08Z" stroke="#ef4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>' : 
        '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M9 12.75L11.25 15L15 9.75M21 12C21 16.9706 16.9706 21 12 21C7.02944 21 3 16.9706 3 12C3 7.02944 7.02944 3 12 3C16.9706 3 21 7.02944 21 12Z" stroke="#10b981" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    
    // Status text
    const statusText = document.createElement('div');
    statusText.style.cssText = `
        font-weight: 600;
    `;
    
    if (data.hasNestedTransaction) {
        statusText.innerHTML = '<span style="color: #ef4444;">Requires Careful Review</span>';
    } else {
        statusText.innerHTML = '<span style="color: #10b981;">Standard Transaction</span>';
    }
    
    statusInfo.appendChild(statusIcon);
    statusInfo.appendChild(statusText);
    banner.appendChild(statusInfo);
    
    return banner;
}

/**
 * Creates a high-level transaction summary
 * @param {Object} data - The formatted data
 * @returns {HTMLElement} Transaction summary element
 */
function createTransactionSummary(data) {
    const summary = document.createElement('div');
    summary.style.cssText = `
        padding: 24px;
        background: var(--bg-subtle);
        border-bottom: 1px solid rgba(0,0,0,0.05);
    `;
    
    // Summary heading
    const heading = document.createElement('h1');
    heading.textContent = 'Transaction Summary';
    heading.style.cssText = `
        font-size: 18px;
        font-weight: 600;
        margin: 0 0 16px 0;
        color: var(--primary-color);
    `;
    summary.appendChild(heading);
    
    // Plain-language summary
    const plainSummary = document.createElement('p');
    plainSummary.style.cssText = `
        font-size: 16px;
        margin: 0 0 16px 0;
        line-height: 1.5;
    `;
    
    // Generate human-readable summary based on transaction type
    let summaryText = "";
    if (data.callDetails && data.callDetails.function) {
        switch (data.callDetails.function) {
            case "transfer":
                if (data.callDetails.parsedData && data.callDetails.parsedData.to && data.callDetails.parsedData.amount) {
                    summaryText = `Send ${data.callDetails.parsedData.amount} to ${formatAddress(data.callDetails.parsedData.to)}`;
                } else {
                    summaryText = "Transfer tokens";
                }
                break;
            case "approve":
                if (data.callDetails.parsedData && data.callDetails.parsedData.spender) {
                    summaryText = `Allow ${formatAddress(data.callDetails.parsedData.spender)} to spend your tokens`;
                } else {
                    summaryText = "Approve token spending";
                }
                break;
            default:
                summaryText = `Call "${data.callDetails.function}" on ${formatAddress(data.callDetails.target)}`;
        }
    } else if (data.basicInfo && data.basicInfo.value && parseFloat(data.basicInfo.value) > 0) {
        summaryText = `Send ${data.basicInfo.value} to ${formatAddress(data.basicInfo.to)}`;
    } else {
        summaryText = "Interact with smart contract";
    }
    
    plainSummary.textContent = summaryText;
    summary.appendChild(plainSummary);
    
    // Warning for nested transactions
    if (data.hasNestedTransaction) {
        const warning = document.createElement('div');
        warning.style.cssText = `
            background: rgba(239,68,68,0.1);
            border-left: 4px solid #ef4444;
            padding: 12px 16px;
            border-radius: 6px;
            font-size: 14px;
            margin-top: 16px;
            line-height: 1.5;
        `;
        warning.innerHTML = '<strong>Caution:</strong> This transaction contains nested operations that will execute additional code. Review all details carefully before approving.';
        summary.appendChild(warning);
    }
    
    return summary;
}

/**
 * Creates the transaction details section
 * @param {Object} data - The formatted data
 * @returns {HTMLElement} Details section element
 */
function createDetailsSection(data) {
    const section = document.createElement('div');
    section.style.cssText = `
        background: #ffffff;
        border-radius: 12px;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        overflow: hidden;
    `;
    
    // Section header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #f9fafb;
        border-bottom: 1px solid rgba(0,0,0,0.05);
        display: flex;
        align-items: center;
        gap: 12px;
    `;
    
    // Header icon
    const headerIcon = document.createElement('div');
    headerIcon.innerHTML = '<svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M10 7V9M10 13H10.01M3 17L5 5L10 3L15 5L17 17L10 19L3 17Z" stroke="#6b7280" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    
    // Header text
    const headerText = document.createElement('h2');
    headerText.textContent = 'Transaction Details';
    headerText.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #374151;
    `;
    
    header.appendChild(headerIcon);
    header.appendChild(headerText);
    section.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 16px 20px;`;
    
    // Create a data table
    const table = document.createElement('div');
    table.style.cssText = `
        display: flex;
        flex-direction: column;
        gap: 12px;
    `;
    
    // Add each transaction detail as a row
    Object.entries(data.basicInfo).forEach(([key, value]) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            align-items: flex-start;
        `;
        
        const keyCell = document.createElement('div');
        keyCell.style.cssText = `
            width: 140px;
            font-size: 14px;
            color: #6b7280;
            flex-shrink: 0;
            padding: 2px 0;
        `;
        
        // Format key name
        const formattedKey = key
            .replace(/([A-Z])/g, ' $1')
            .replace(/_/g, ' ')
            .replace(/^./, str => str.toUpperCase());
            
        keyCell.textContent = formattedKey;
        
        const valueCell = document.createElement('div');
        valueCell.style.cssText = `
            flex: 1;
            font-size: 14px;
            word-break: break-all;
            padding: 2px 0;
            text-align: left;
        `;
        
        // Format value display
        if (typeof value === 'string' && value.includes(' ✅')) {
            const parts = value.split(' (');
            const addressEl = document.createElement('div');
            addressEl.textContent = parts[0];
            addressEl.style.cssText = 'font-family: monospace;';
            
            valueCell.appendChild(addressEl);
            
            if (parts.length > 1) {
                const verifiedEl = document.createElement('div');
                verifiedEl.innerHTML = `<span style="color: #10b981; font-weight: 500;">\u2713 ${parts[1].replace('✅)', '')}</span>`;
                valueCell.appendChild(verifiedEl);
            }
        } else {
            valueCell.textContent = value;
        }
        
        row.appendChild(keyCell);
        row.appendChild(valueCell);
        table.appendChild(row);
    });
    
    content.appendChild(table);
    section.appendChild(content);
    
    return section;
}

/**
 * Creates the function call section
 * @param {Object} callDetails - The function call details
 * @returns {HTMLElement} Function call section element
 */
function createFunctionCallSection(callDetails) {
    const section = document.createElement('div');
    section.style.cssText = `
        background: #ffffff;
        border-radius: 12px;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        overflow: hidden;
    `;
    
    // Section header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #f9fafb;
        border-bottom: 1px solid rgba(0,0,0,0.05);
        display: flex;
        align-items: center;
        gap: 12px;
    `;
    
    // Header icon
    const headerIcon = document.createElement('div');
    headerIcon.innerHTML = '<svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 4H12V12H4V4Z M8 8H16V16H8V8Z" stroke="#6b7280" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    
    // Header text
    const headerText = document.createElement('h2');
    headerText.textContent = 'Function Call';
    headerText.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #374151;
    `;
    
    header.appendChild(headerIcon);
    header.appendChild(headerText);
    section.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 16px 20px;`;
    
    // Function signature
    const functionSignature = document.createElement('div');
    functionSignature.style.cssText = `
        background: #f9fafb;
        border-radius: 8px;
        padding: 12px 16px;
        margin-bottom: 16px;
        font-family: monospace;
        font-size: 14px;
        color: #4f46e5;
    `;
    functionSignature.textContent = callDetails.function || 'Unknown Function';
    content.appendChild(functionSignature);
    
    // Target contract section
    const targetSection = document.createElement('div');
    targetSection.style.cssText = 'margin-bottom: 20px;';
    
    const targetLabel = document.createElement('div');
    targetLabel.textContent = 'Target Contract:';
    targetLabel.style.cssText = `
        font-size: 14px;
        font-weight: 500;
        margin-bottom: 8px;
        color: #374151;
    `;
    
    const targetValue = document.createElement('div');
    targetValue.style.cssText = `
        background: #f9fafb;
        padding: 12px;
        border-radius: 8px;
        font-family: monospace;
        font-size: 14px;
        word-break: break-all;
    `;
    
    // Check if the target has a verified name
    if (typeof callDetails.target === 'string' && callDetails.target.includes(' ✅')) {
        const parts = callDetails.target.split(' (');
        targetValue.textContent = parts[0];
        
        if (parts.length > 1) {
            const verifiedName = document.createElement('div');
            verifiedName.innerHTML = `<span style="color: #10b981; font-weight: 500;">\u2713 ${parts[1].replace('✅)', '')}</span>`;
            targetValue.appendChild(verifiedName);
        }
    } else {
        targetValue.textContent = callDetails.target || 'Unknown Contract';
    }
    
    targetSection.appendChild(targetLabel);
    targetSection.appendChild(targetValue);
    content.appendChild(targetSection);
    
    // Parameters section (if any)
    if (callDetails.parsedData && Object.keys(callDetails.parsedData).length > 0) {
        const paramsSection = document.createElement('div');
        
        const paramsLabel = document.createElement('div');
        paramsLabel.textContent = 'Parameters:';
        paramsLabel.style.cssText = `
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 8px;
            color: #374151;
        `;
        
        paramsSection.appendChild(paramsLabel);
        
        const paramsTable = document.createElement('div');
        paramsTable.style.cssText = `
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            overflow: hidden;
        `;
        
        Object.entries(callDetails.parsedData).forEach(([key, value], index) => {
            const row = document.createElement('div');
            row.style.cssText = `
                display: flex;
                ${index < Object.keys(callDetails.parsedData).length - 1 ? 'border-bottom: 1px solid #e5e7eb;' : ''}
            `;
            
            const keyCell = document.createElement('div');
            keyCell.textContent = key;
            keyCell.style.cssText = `
                width: 120px;
                padding: 12px;
                background: #f9fafb;
                color: #4b5563;
                font-family: monospace;
                border-right: 1px solid #e5e7eb;
                font-size: 14px;
            `;
            
            const valueCell = document.createElement('div');
            valueCell.style.cssText = `
                flex: 1;
                padding: 12px;
                word-break: break-all;
                font-family: monospace;
                font-size: 14px;
            `;
            
            // Special formatting for addresses with verification status
            if (typeof value === 'string' && value.includes(' ✅')) {
                const parts = value.split(' (');
                valueCell.textContent = parts[0];
                
                if (parts.length > 1) {
                    const verifiedInfo = document.createElement('div');
                    verifiedInfo.innerHTML = `<span style="color: #10b981; font-weight: 500;">\u2713 ${parts[1].replace('✅)', '')}</span>`;
                    valueCell.appendChild(verifiedInfo);
                }
            } else {
                valueCell.textContent = value;
            }
            
            row.appendChild(keyCell);
            row.appendChild(valueCell);
            paramsTable.appendChild(row);
        });
        
        paramsSection.appendChild(paramsTable);
        content.appendChild(paramsSection);
    }
    
    section.appendChild(content);
    return section;
}

/**
 * Creates the nested transaction section
 * @param {Object} nestedInfo - The nested transaction information
 * @returns {HTMLElement} Nested transaction section element
 */
function createNestedTransactionSection(nestedInfo) {
    const section = document.createElement('div');
    section.style.cssText = `
        background: #fff5f5;
        border-radius: 12px;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        overflow: hidden;
        border: 1px solid rgba(239,68,68,0.3);
    `;
    
    // Section header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: rgba(239,68,68,0.1);
        border-bottom: 1px solid rgba(239,68,68,0.2);
        display: flex;
        align-items: center;
        gap: 12px;
    `;
    
    // Header icon
    const headerIcon = document.createElement('div');
    headerIcon.innerHTML = '<svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M8.256 3.845a2.001 2.001 0 0 1 3.488 0c.311.627.95 1.074 1.675 1.078a1.999 1.999 0 0 1 1.743 3.022A2 2 0 0 0 15.87 10.5a2 2 0 0 0-.708 2.555 2 2 0 0 1-1.743 3.022c-.725.004-1.364.45-1.674 1.078a2 2 0 0 1-3.488 0c-.311-.627-.95-1.074-1.675-1.078a2 2 0 0 1-1.743-3.022A2 2 0 0 0 4.13 10.5a2 2 0 0 0 .708-2.555 2 2 0 0 1 1.743-3.022c.725-.004 1.364-.45 1.674-1.078zM10 11a1 1 0 1 0 0-2 1 1 0 0 0 0 2z" stroke="#ef4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    
    // Header text
    const headerText = document.createElement('h2');
    headerText.textContent = 'Nested Transaction';
    headerText.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #ef4444;
    `;
    
    header.appendChild(headerIcon);
    header.appendChild(headerText);
    section.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 16px 20px;`;
    
    // Warning message
    const warning = document.createElement('div');
    warning.style.cssText = `
        margin-bottom: 20px;
        background: rgba(239,68,68,0.08);
        border-radius: 8px;
        padding: 12px 16px;
        line-height: 1.5;
        font-size: 14px;
    `;
    warning.innerHTML = '<strong>High-Risk Functionality:</strong> This transaction will execute additional code that could perform multiple operations. Always verify that you trust the contract and understand all operations it will perform.';
    content.appendChild(warning);
    
    // Nested details table
    const detailsTable = document.createElement('div');
    detailsTable.style.cssText = `
        background: rgba(255,255,255,0.7);
        border-radius: 8px;
        overflow: hidden;
        margin-bottom: 20px;
        border: 1px solid rgba(239,68,68,0.2);
    `;
    
    // Add rows for each piece of nested transaction info
    const info = [
        { key: 'Safe Address', value: nestedInfo.safe },
        { key: 'Nonce', value: nestedInfo.nonce },
        { key: 'Hash', value: nestedInfo.hash }
    ];
    
    info.forEach(({ key, value }, index) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            ${index < info.length - 1 ? 'border-bottom: 1px solid rgba(239,68,68,0.2);' : ''}
        `;
        
        const keyCell = document.createElement('div');
        keyCell.textContent = key;
        keyCell.style.cssText = `
            width: 140px;
            padding: 12px 16px;
            background: rgba(239,68,68,0.05);
            color: #b91c1c;
            font-weight: 500;
            flex-shrink: 0;
            font-size: 14px;
        `;
        
        const valueCell = document.createElement('div');
        valueCell.style.cssText = `
            flex: 1;
            padding: 12px 16px;
            word-break: break-all;
            font-family: monospace;
            font-size: 14px;
        `;
        
        // Special formatting for addresses with verification status
        if (typeof value === 'string' && value.includes(' ✅')) {
            const parts = value.split(' (');
            valueCell.textContent = parts[0];
            
            if (parts.length > 1) {
                const verifiedInfo = document.createElement('div');
                verifiedInfo.innerHTML = `<span style="color: #10b981; font-weight: 500;">\u2713 ${parts[1].replace('✅)', '')}</span>`;
                valueCell.appendChild(verifiedInfo);
            }
        } else {
            valueCell.textContent = value;
        }
        
        row.appendChild(keyCell);
        row.appendChild(valueCell);
        detailsTable.appendChild(row);
    });
    
    content.appendChild(detailsTable);
    
    // Nested function call if available
    if (nestedInfo.callDetails) {
        const nestedFunctionTitle = document.createElement('h3');
        nestedFunctionTitle.textContent = 'Nested Function Call';
        nestedFunctionTitle.style.cssText = `
            margin: 0 0 12px 0;
            font-size: 15px;
            font-weight: 600;
            color: #b91c1c;
        `;
        content.appendChild(nestedFunctionTitle);
        
        // Function signature
        const functionSignature = document.createElement('div');
        functionSignature.style.cssText = `
            background: rgba(255,255,255,0.7);
            border: 1px solid rgba(239,68,68,0.2);
            border-radius: 8px;
            padding: 12px 16px;
            margin-bottom: 12px;
            font-family: monospace;
            font-size: 14px;
            color: #4f46e5;
        `;
        functionSignature.textContent = nestedInfo.callDetails.function || 'Unknown Function';
        content.appendChild(functionSignature);
        
        // Target contract if available
        if (nestedInfo.callDetails.target) {
            const targetLabel = document.createElement('div');
            targetLabel.textContent = 'Target:';
            targetLabel.style.cssText = `
                font-size: 14px;
                font-weight: 500;
                margin-bottom: 8px;
                color: #374151;
            `;
            
            const targetValue = document.createElement('div');
            targetValue.style.cssText = `
                background: rgba(255,255,255,0.7);
                border: 1px solid rgba(239,68,68,0.2);
                padding: 12px;
                border-radius: 8px;
                word-break: break-all;
                font-family: monospace;
                font-size: 14px;
            `;
            targetValue.textContent = nestedInfo.callDetails.target;
            
            content.appendChild(targetLabel);
            content.appendChild(targetValue);
        }
    }
    
    section.appendChild(content);
    return section;
}

/**
 * Creates the security verification section
 * @param {Object} hashes - The transaction hashes
 * @returns {HTMLElement} Security section element
 */
function createSecuritySection(hashes) {
    const section = document.createElement('div');
    section.style.cssText = `
        background: #ffffff;
        border-radius: 12px;
        box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        overflow: hidden;
    `;
    
    // Section header
    const header = document.createElement('div');
    header.style.cssText = `
        padding: 16px 20px;
        background: #f9fafb;
        border-bottom: 1px solid rgba(0,0,0,0.05);
        display: flex;
        align-items: center;
        gap: 12px;
    `;
    
    // Header icon
    const headerIcon = document.createElement('div');
    headerIcon.innerHTML = '<svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M5 10.0001H15M5 10.0001C3.89543 10.0001 3 9.10463 3 8.00006V6.00006C3 4.89549 3.89543 4.00006 5 4.00006H15C16.1046 4.00006 17 4.89549 17 6.00006V8.00006C17 9.10463 16.1046 10.0001 15 10.0001M5 10.0001C3.89543 10.0001 3 10.8955 3 12.0001V14.0001C3 15.1046 3.89543 16.0001 5 16.0001H15C16.1046 16.0001 17 15.1046 17 14.0001V12.0001C17 10.8955 16.1046 10.0001 15 10.0001M10 7.00006V7.00998M10 13.0001V13.01" stroke="#6b7280" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>';
    
    // Header text
    const headerText = document.createElement('h2');
    headerText.textContent = 'Hardware Wallet Verification';
    headerText.style.cssText = `
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #374151;
    `;
    
    header.appendChild(headerIcon);
    header.appendChild(headerText);
    section.appendChild(header);
    
    // Content
    const content = document.createElement('div');
    content.style.cssText = `padding: 16px 20px;`;
    
    // Verification guidance
    const guidance = document.createElement('div');
    guidance.style.cssText = `
        background: #f3f4f6;
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 24px;
        border-left: 4px solid #10b981;
    `;
    
    guidance.innerHTML = `
        <p style="margin: 0 0 12px 0; font-weight: 600; font-size: 15px; color: #374151;">Verification Checklist:</p>
        <ol style="margin: 0; padding-left: 24px; font-size: 14px; line-height: 1.6; color: #4b5563; text-align: left;">
            <li>Check that all transaction details match your expectations</li>
            <li>Verify all contract addresses are correct and verified</li>
            <li>Confirm the hash values below match exactly what appears on your hardware wallet</li>
            <li>If this transaction contains nested operations, review them with extra caution</li>
        </ol>
    `;
    
    content.appendChild(guidance);
    
    // Hash values section
    const hashesTitle = document.createElement('div');
    hashesTitle.textContent = 'Compare these values with your hardware wallet:';
    hashesTitle.style.cssText = `
        font-size: 14px;
        font-weight: 500;
        margin-bottom: 16px;
        color: #374151;
    `;
    content.appendChild(hashesTitle);
    
    // Hash boxes
    Object.entries(hashes).forEach(([key, value]) => {
        const hashContainer = document.createElement('div');
        hashContainer.style.cssText = 'margin-bottom: 16px;';
        
        const hashLabel = document.createElement('div');
        hashLabel.textContent = key.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase());
        hashLabel.style.cssText = `
            font-size: 13px;
            margin-bottom: 8px;
            color: #6b7280;
        `;
        
        const hashBox = document.createElement('div');
        hashBox.style.cssText = `
            background: #f9fafb;
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            padding: 12px 16px;
            position: relative;
        `;
        
        const hashText = document.createElement('code');
        hashText.textContent = value;
        hashText.style.cssText = `
            font-family: monospace;
            color: #1f2937;
            font-size: 14px;
            word-break: break-all;
        `;
        
        hashBox.appendChild(hashText);
        
        hashContainer.appendChild(hashLabel);
        hashContainer.appendChild(hashBox);
        content.appendChild(hashContainer);
    });
    
    section.appendChild(content);
    return section;
}

/**
 * Helper function to format addresses with ellipsis in the middle
 * @param {string} address - The address to format
 * @returns {string} Formatted address
 */
function formatAddress(address) {
    if (!address || typeof address !== 'string') return 'Unknown Address';
    
    // Handle addresses with verification info
    if (address.includes(' (')) {
        const parts = address.split(' (');
        const addr = parts[0];
        const name = parts[1].replace('✅)', '');
        
        if (addr.length > 12) {
            return `${addr.substring(0, 6)}...${addr.substring(addr.length - 4)} (${name})`;
        }
        return address;
    }
    
    // Regular address
    if (address.length > 12) {
        return `${address.substring(0, 6)}...${address.substring(address.length - 4)}`;
    }
    return address;
}