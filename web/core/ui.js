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
      background: #111827;
      color: #e5e7eb;
      border-radius: 16px;
      box-shadow: 0 10px 25px rgba(0,0,0,0.3);
      max-width: 960px;
      margin: 0 auto;
      overflow: hidden;
      --primary-color: #94a3b8;
      --bg-subtle:rgb(52, 51, 51);
      --bg-strong:rgb(50, 47, 47);
      --border-color:rgb(47, 45, 45);
      border: 1px solid var(--border-color);
  `;
  
  // Status banner
  container.appendChild(createStatusBanner(formattedData));
  
  // Transaction summary
  container.appendChild(createTransactionSummary(formattedData));
  
  // Main content area
  const contentArea = document.createElement('div');
  contentArea.style.cssText = `
      padding: 28px;
      display: flex;
      flex-direction: column;
      gap: 28px;
      background: #1f2937;
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
      border-bottom: 2px solid var(--border-color);
  `;
  
  // Status information
  const statusInfo = document.createElement('div');
  statusInfo.style.cssText = `
      display: flex;
      align-items: center;
  `;
  
  // Status text
  const statusText = document.createElement('div');
  statusText.style.cssText = `
      font-weight: 700;
      font-size: 18px;
  `;
  
  if (data.hasNestedTransaction) {
      statusText.innerHTML = '<span style="color: #cbd5e1;">NESTED TRANSACTION</span>';
  } else {
      statusText.innerHTML = '<span style="color: #34d399;">STANDARD TRANSACTION</span>';
  }
  
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
      border-bottom: 1px solid rgba(255,255,255,0.1);
  `;
  
  // Summary heading
  const heading = document.createElement('h1');
  heading.textContent = 'Transaction Summary';
  heading.style.cssText = `
      font-size: 20px;
      font-weight: 700;
      margin: 0 0 16px 0;
      color: var(--primary-color);
  `;
  
  summary.appendChild(heading);
  
  // Plain-language summary
  const plainSummary = document.createElement('div');
  plainSummary.style.cssText = `
      font-size: 16px;
      margin: 0 0 20px 0;
      line-height: 1.5;
      padding: 16px;
      background: #111827;
      border-radius: 8px;
      border: 1px solid ${data.hasNestedTransaction ? 'rgba(148,163,184,0.3)' : 'rgba(16,185,129,0.3)'};
  `;
  
  // Generate human-readable summary based on transaction type
  let summaryText = "";
  if (data.callDetails && data.callDetails.functionName) {
      switch (data.callDetails.functionName) {
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
          case "unknown":
              summaryText = `Execute an unknown function on ${formatAddress(data.callDetails.target)}`;
              break;
          default:
              summaryText = `Call "${data.callDetails.functionName}" on ${formatAddress(data.callDetails.target)}`;
      }
  } else if (data.basicInfo && data.basicInfo.value && parseFloat(data.basicInfo.value) > 0) {
      summaryText = `Send ${data.basicInfo.value} to ${formatAddress(data.basicInfo.target)}`;
  } else {
      summaryText = "Interact with smart contract";
  }
  
  plainSummary.textContent = summaryText;
  summary.appendChild(plainSummary);
  
  // Information for nested transactions
  if (data.hasNestedTransaction) {
      const nestedInfo = document.createElement('div');
      nestedInfo.style.cssText = `
          background: #1f2937;
          border-left: 4px solid #94a3b8;
          padding: 16px;
          border-radius: 6px;
          font-size: 15px;
          margin-top: 16px;
          line-height: 1.5;
      `;
      
      nestedInfo.innerHTML = `
        <strong style="color: #cbd5e1; font-size: 16px; display: block; margin-bottom: 4px;">Contains Nested Transaction(s)</strong>
        This transaction causes another Safe to execute a transaction.
      `;
      
      summary.appendChild(nestedInfo);
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
      background: #111827;
      border-radius: 12px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.2);
      overflow: hidden;
      border: 1px solid #374151;
  `;
  
  // Section header
  const header = document.createElement('div');
  header.style.cssText = `
      padding: 16px 20px;
      background: #1f2937;
      border-bottom: 1px solid #374151;
  `;
  
  // Header text
  const headerText = document.createElement('h2');
  headerText.textContent = 'Transaction Details';
  headerText.style.cssText = `
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #e5e7eb;
  `;
  
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
          color: #9ca3af;
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
          color: #e5e7eb;
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
              verifiedEl.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
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
      background: #111827;
      border-radius: 12px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.2);
      overflow: hidden;
      border: 1px solid #374151;
  `;
  
  // Section header
  const header = document.createElement('div');
  header.style.cssText = `
      padding: 16px 20px;
      background: #1f2937;
      border-bottom: 1px solid #374151;
  `;
  
  // Header text
  const headerText = document.createElement('h2');
  headerText.textContent = 'Function Call';
  headerText.style.cssText = `
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #e5e7eb;
  `;
  
  header.appendChild(headerText);
  section.appendChild(header);
  
  // Content
  const content = document.createElement('div');
  content.style.cssText = `padding: 16px 20px;`;
  
  // Function signature
  const functionSignature = document.createElement('div');
  
  const isUnknownFunction = callDetails.functionName === 'unknown';
  
  functionSignature.style.cssText = `
      background: ${isUnknownFunction ? '#291e21' : '#1e293b'};
      border-radius: 8px;
      padding: 12px 16px;
      margin-bottom: 16px;
      font-family: monospace;
      font-size: 14px;
      color: ${isUnknownFunction ? '#ef4444' : '#d1d5db'};
      border: ${isUnknownFunction ? '1px solid #dc2626' : '1px solid #374151'};
  `;
  
  // Add function name
  functionSignature.textContent = callDetails.functionName || 'Unknown Function';
  
  content.appendChild(functionSignature);
  
  // Add warning when unknown function
  if (isUnknownFunction) {
    const warningBox = document.createElement('div');
    warningBox.style.cssText = `
      background: #291e21;
      border-left: 4px solid #ef4444;
      padding: 12px 16px;
      border-radius: 6px;
      margin-bottom: 16px;
      font-size: 14px;
      color: #e5e7eb;
    `;
    warningBox.innerHTML = '<strong>Caution:</strong> This transaction contains an unknown function. Review the raw calldata carefully before approving.';
    content.appendChild(warningBox);
  }
  
  // Target contract section
  const targetSection = document.createElement('div');
  targetSection.style.cssText = 'margin-bottom: 20px;';
  
  const targetLabel = document.createElement('div');
  targetLabel.textContent = 'Target Contract:';
  targetLabel.style.cssText = `
      font-size: 14px;
      font-weight: 500;
      margin-bottom: 8px;
      color: #9ca3af;
  `;
  
  const targetValue = document.createElement('div');
  targetValue.style.cssText = `
      background: #1f2937;
      padding: 12px;
      border-radius: 8px;
      font-family: monospace;
      font-size: 14px;
      word-break: break-all;
      display: flex;
      flex-direction: column;
      gap: 4px;
      color: #e5e7eb;
      border: 1px solid #374151;
  `;
  
  // Check if the target has a verified name
  if (typeof callDetails.target === 'string' && callDetails.target.includes(' ✅')) {
      const parts = callDetails.target.split(' (');
      targetValue.textContent = parts[0];
      
      if (parts.length > 1) {
          const verifiedName = document.createElement('div');
          verifiedName.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
          targetValue.appendChild(verifiedName);
      }
  } else {
      const addressDisplay = document.createElement('div');
      addressDisplay.textContent = callDetails.target || 'Unknown Contract';
      
      targetValue.appendChild(addressDisplay);
      
      if (callDetails.targetName) {
          const verifiedName = document.createElement('div');
          verifiedName.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${callDetails.targetName}</span>`;
          targetValue.appendChild(verifiedName);
      }
  }
  
  targetSection.appendChild(targetLabel);
  targetSection.appendChild(targetValue);
  content.appendChild(targetSection);
  
  // Raw data section for unknown functions
  if (callDetails.rawData) {
    const rawDataSection = document.createElement('div');
    
    const rawDataLabel = document.createElement('div');
    rawDataLabel.textContent = 'Raw Calldata:';
    rawDataLabel.style.cssText = `
        font-size: 14px;
        font-weight: 500;
        margin-bottom: 8px;
        color: #9ca3af;
    `;
    
    rawDataSection.appendChild(rawDataLabel);
    
    const rawDataValue = document.createElement('div');
    rawDataValue.style.cssText = `
        background: ${isUnknownFunction ? '#291e21' : '#1f2937'};
        padding: 12px;
        border-radius: 8px;
        font-family: monospace;
        font-size: 14px;
        word-break: break-all;
        max-height: 200px;
        overflow-y: auto;
        border: 1px solid ${isUnknownFunction ? '#dc2626' : '#374151'};
        color: #e5e7eb;
    `;
    rawDataValue.textContent = callDetails.rawData;
    
    rawDataSection.appendChild(rawDataValue);
    content.appendChild(rawDataSection);
  }
  
  // Parameters section (if any)
  if (callDetails.parsedData && Object.keys(callDetails.parsedData).length > 0) {
      const paramsSection = document.createElement('div');
      
      const paramsLabel = document.createElement('div');
      paramsLabel.textContent = 'Parameters:';
      paramsLabel.style.cssText = `
          font-size: 14px;
          font-weight: 500;
          margin-bottom: 8px;
          color: #9ca3af;
      `;
      
      paramsSection.appendChild(paramsLabel);
      
      const paramsTable = document.createElement('div');
      paramsTable.style.cssText = `
          border: 1px solid #374151;
          border-radius: 8px;
          overflow: hidden;
      `;
      
      Object.entries(callDetails.parsedData).forEach(([key, value], index) => {
          const row = document.createElement('div');
          row.style.cssText = `
              display: flex;
              ${index < Object.keys(callDetails.parsedData).length - 1 ? 'border-bottom: 1px solid #374151;' : ''}
          `;
          
          const keyCell = document.createElement('div');
          keyCell.textContent = key;
          keyCell.style.cssText = `
              width: 120px;
              padding: 12px;
              background: #1f2937;
              color: #9ca3af;
              font-family: monospace;
              border-right: 1px solid #374151;
              font-size: 14px;
          `;
          
          const valueCell = document.createElement('div');
          valueCell.style.cssText = `
              flex: 1;
              padding: 12px;
              word-break: break-all;
              font-family: monospace;
              font-size: 14px;
              color: #e5e7eb;
              background: #111827;
          `;
          
          // Special formatting for addresses with verification status
          if (typeof value === 'string' && value.includes(' ✅')) {
              const parts = value.split(' (');
              valueCell.textContent = parts[0];
              
              if (parts.length > 1) {
                  const verifiedInfo = document.createElement('div');
                  verifiedInfo.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
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
      background: #111827;
      border-radius: 12px;
      box-shadow: 0 2px 4px rgba(0,0,0,0.2);
      overflow: hidden;
      border: 1px solid #374151;
  `;
  
  // Section header
  const header = document.createElement('div');
  header.style.cssText = `
      padding: 16px 20px;
      background: #1f2937;
      border-bottom: 1px solid #374151;
  `;
  
  // Header text
  const headerText = document.createElement('h2');
  headerText.textContent = 'Nested Transaction';
  headerText.style.cssText = `
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #e5e7eb;
  `;
  
  header.appendChild(headerText);
  section.appendChild(header);
  
  // Content
  const content = document.createElement('div');
  content.style.cssText = `padding: 20px;`;
  
  // Information message
  const info = document.createElement('div');
  info.style.cssText = `
      margin-bottom: 24px;
      background: #1f2937;
      border-left: 4px solid #94a3b8;
      padding: 16px;
      border-radius: 6px;
      font-size: 15px;
      line-height: 1.5;
      color: #e5e7eb;
  `;
  info.innerHTML = `
    <div style="font-weight: 600; font-size: 16px; color: #cbd5e1; margin-bottom: 8px;">Nested Transaction Details</div>
    <p style="margin: 0 0 12px 0;">This transaction contains nested operations that will execute additional code. These operations are shown below so you can review them before approving.</p>
  `;
  content.appendChild(info);
  
  // Nested details table
  const detailsTable = document.createElement('div');
  detailsTable.style.cssText = `
      background: #1f2937;
      border-radius: 8px;
      overflow: hidden;
      margin-bottom: 24px;
      border: 1px solid #374151;
  `;
  
  // Add rows for each piece of nested transaction info
  const detailsInfo = [
      { key: 'Safe Address', value: nestedInfo.safe },
      { key: 'Nonce', value: nestedInfo.nonce },
      { key: 'Hash', value: nestedInfo.hash }
  ];
  
  detailsInfo.forEach(({ key, value }, index) => {
      const row = document.createElement('div');
      row.style.cssText = `
          display: flex;
          ${index < detailsInfo.length - 1 ? 'border-bottom: 1px solid #374151;' : ''}
      `;
      
      const keyCell = document.createElement('div');
      keyCell.textContent = key;
      keyCell.style.cssText = `
          width: 140px;
          padding: 12px 16px;
          background: #111827;
          color: #9ca3af;
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
          color: #e5e7eb;
      `;
      
      // Special formatting for addresses with verification status
      if (typeof value === 'string' && value.includes(' ✅')) {
          const parts = value.split(' (');
          valueCell.textContent = parts[0];
          
          if (parts.length > 1) {
              const verifiedInfo = document.createElement('div');
              verifiedInfo.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
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
      nestedFunctionTitle.textContent = 'Nested Function Call Details';
      nestedFunctionTitle.style.cssText = `
          margin: 0 0 12px 0;
          font-size: 16px;
          font-weight: 600;
          color: #e5e7eb;
      `;
      
      content.appendChild(nestedFunctionTitle);
      
      // Function signature
      const functionSignature = document.createElement('div');
      functionSignature.style.cssText = `
          background: #1e293b;
          border: 1px solid #374151;
          border-radius: 8px;
          padding: 12px 16px;
          margin-bottom: 16px;
          font-family: monospace;
          font-size: 14px;
          color: #d1d5db;
          font-weight: 600;
      `;
      
      const isUnknownFunction = nestedInfo.callDetails.functionName === 'unknown';
      
      if (isUnknownFunction) {
        functionSignature.textContent = 'Unknown Function';
        functionSignature.style.color = '#ef4444';
        functionSignature.style.background = '#291e21';
        functionSignature.style.border = '1px solid #dc2626';
      } else {
        functionSignature.textContent = nestedInfo.callDetails.functionName || 'Unknown Function';
      }
      
      content.appendChild(functionSignature);
      
      // Target contract if available
      if (nestedInfo.callDetails.target) {
          const targetLabel = document.createElement('div');
          targetLabel.textContent = 'Target Contract:';
          targetLabel.style.cssText = `
              font-size: 14px;
              font-weight: 500;
              margin-bottom: 8px;
              color: #9ca3af;
          `;
          
          const targetValue = document.createElement('div');
          targetValue.style.cssText = `
              background: #1f2937;
              border: 1px solid #374151;
              padding: 12px;
              border-radius: 8px;
              word-break: break-all;
              font-family: monospace;
              font-size: 14px;
              display: flex;
              flex-direction: column;
              gap: 4px;
              color: #e5e7eb;
          `;
          
          // Check if the target has a verified name
          if (typeof nestedInfo.callDetails.target === 'string' && nestedInfo.callDetails.target.includes(' ✅')) {
              const parts = nestedInfo.callDetails.target.split(' (');
              const addressDisplay = document.createElement('div');
              addressDisplay.textContent = parts[0];
              targetValue.appendChild(addressDisplay);
              
              if (parts.length > 1) {
                  const verifiedName = document.createElement('div');
                  verifiedName.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
                  targetValue.appendChild(verifiedName);
              }
          } else {
              const addressDisplay = document.createElement('div');
              addressDisplay.textContent = nestedInfo.callDetails.target;
              targetValue.appendChild(addressDisplay);
              
              if (nestedInfo.callDetails.targetName) {
                  const verifiedName = document.createElement('div');
                  verifiedName.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${nestedInfo.callDetails.targetName}</span>`;
                  targetValue.appendChild(verifiedName);
              }
          }
          
          content.appendChild(targetLabel);
          content.appendChild(targetValue);
      }
      
      // Raw data if available for unknown function
      if (nestedInfo.callDetails.rawData) {
        const rawDataSection = document.createElement('div');
        
        const rawDataLabel = document.createElement('div');
        rawDataLabel.textContent = 'Raw Calldata:';
        rawDataLabel.style.cssText = `
            font-size: 14px;
            font-weight: 500;
            margin: 16px 0 8px 0;
            color: #9ca3af;
        `;
        
        rawDataSection.appendChild(rawDataLabel);
        
        const rawDataValue = document.createElement('div');
        rawDataValue.style.cssText = `
            background: ${isUnknownFunction ? '#291e21' : '#1f2937'};
            padding: 12px;
            border-radius: 8px;
            font-family: monospace;
            font-size: 14px;
            word-break: break-all;
            max-height: 200px;
            overflow-y: auto;
            border: 1px solid ${isUnknownFunction ? '#dc2626' : '#374151'};
            color: #e5e7eb;
        `;
        rawDataValue.textContent = nestedInfo.callDetails.rawData;
        
        rawDataSection.appendChild(rawDataValue);
        content.appendChild(rawDataSection);
      }
      
      // Parameters section (if any)
      if (nestedInfo.callDetails.parsedData && Object.keys(nestedInfo.callDetails.parsedData).length > 0) {
          const paramsSection = document.createElement('div');
          
          const paramsLabel = document.createElement('div');
          paramsLabel.textContent = 'Parameters:';
          paramsLabel.style.cssText = `
              font-size: 14px;
              font-weight: 500;
              margin: 16px 0 8px 0;
              color: #9ca3af;
          `;
          
          paramsSection.appendChild(paramsLabel);
          
          const paramsTable = document.createElement('div');
          paramsTable.style.cssText = `
              border: 1px solid #374151;
              border-radius: 8px;
              overflow: hidden;
              background: #1f2937;
          `;
          
          Object.entries(nestedInfo.callDetails.parsedData).forEach(([key, value], index) => {
              const row = document.createElement('div');
              row.style.cssText = `
                  display: flex;
                  ${index < Object.keys(nestedInfo.callDetails.parsedData).length - 1 ? 'border-bottom: 1px solid #374151;' : ''}
              `;
              
              const keyCell = document.createElement('div');
              keyCell.textContent = key;
              keyCell.style.cssText = `
                  width: 120px;
                  padding: 12px;
                  background: #111827;
                  color: #9ca3af;
                  font-family: monospace;
                  border-right: 1px solid #374151;
                  font-size: 14px;
              `;
              
              const valueCell = document.createElement('div');
              valueCell.style.cssText = `
                  flex: 1;
                  padding: 12px;
                  word-break: break-all;
                  font-family: monospace;
                  font-size: 14px;
                  color: #e5e7eb;
              `;
              
              if (typeof value === 'string' && value.includes(' ✅')) {
                  const parts = value.split(' (');
                  valueCell.textContent = parts[0];
                  
                  if (parts.length > 1) {
                      const verifiedInfo = document.createElement('div');
                      verifiedInfo.innerHTML = `<span style="color: #94a3b8; font-weight: 500;">[Identified] ${parts[1].replace('✅)', '')}</span>`;
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
      background: #111827;
      border-radius: 12px;
      box-shadow: 0 2px 6px rgba(0,0,0,0.2);
      overflow: hidden;
      border: 1px solid #374151;
  `;
  
  // Section header
  const header = document.createElement('div');
  header.style.cssText = `
      padding: 16px 20px;
      background: #1f2937;
      border-bottom: 1px solid #374151;
  `;
  
  // Header text
  const headerText = document.createElement('h2');
  headerText.textContent = 'Hardware Wallet Verification';
  headerText.style.cssText = `
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #e5e7eb;
  `;
  
  header.appendChild(headerText);
  section.appendChild(header);
  
  // Content
  const content = document.createElement('div');
  content.style.cssText = `padding: 20px;`;
  
  // Verification guidance
  const guidance = document.createElement('div');
  guidance.style.cssText = `
      background: #1f2937;
      border-radius: 8px;
      padding: 20px;
      margin-bottom: 24px;
      border-left: 4px solid #94a3b8;
      color: #e5e7eb;
  `;
  
  guidance.innerHTML = `
      <div style="font-weight: 700; font-size: 16px; color: #cbd5e1; margin-bottom: 16px;">Verification Checklist</div>
      <ol style="margin: 0; padding-left: 24px; font-size: 14px; line-height: 1.6; color: #d1d5db; text-align: left;">
          <li><strong>Compare the hashes</strong> below with what appears on your hardware wallet</li>
          <li>Verify all contract addresses are correct</li>
          <li>Make sure transaction details match your expectations</li>
      </ol>
  `;
  
  content.appendChild(guidance);
  
  // Hash values section
  const hashesTitle = document.createElement('div');
  hashesTitle.textContent = 'Compare these values with your hardware wallet:';
  hashesTitle.style.cssText = `
      font-size: 15px;
      font-weight: 600;
      margin-bottom: 16px;
      color: #e5e7eb;
  `;
  content.appendChild(hashesTitle);
  
  // Hash boxes - display in a grid for better visual grouping
  const hashesContainer = document.createElement('div');
  hashesContainer.style.cssText = `
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 16px;
  `;
  
  Object.entries(hashes).forEach(([key, value]) => {
      const hashContainer = document.createElement('div');
      hashContainer.style.cssText = `
          background: #1f2937;
          border: 1px solid #374151;
          border-radius: 8px;
          overflow: hidden;
      `;
      
      const hashLabel = document.createElement('div');
      hashLabel.textContent = key.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase());
      hashLabel.style.cssText = `
          font-size: 13px;
          padding: 8px 12px;
          background: #111827;
          color: #9ca3af;
          font-weight: 600;
          border-bottom: 1px solid #374151;
      `;
      
      const hashValue = document.createElement('div');
      hashValue.style.cssText = `padding: 12px;`;
      
      const hashText = document.createElement('code');
      hashText.textContent = value;
      hashText.style.cssText = `
          font-family: monospace;
          color: #d1d5db;
          font-size: 14px;
          word-break: break-all;
          display: block;
      `;
      
      hashValue.appendChild(hashText);
      
      hashContainer.appendChild(hashLabel);
      hashContainer.appendChild(hashValue);
      hashesContainer.appendChild(hashContainer);
  });
  
  content.appendChild(hashesContainer);
  
  // Final warning about verification
  const finalWarning = document.createElement('div');
  finalWarning.style.cssText = `
    margin-top: 24px;
    padding: 12px 16px;
    border-radius: 8px;
    background: #1f2937;
    border: 1px solid #4b5563;
    color: #e5e7eb;
    font-size: 14px;
    line-height: 1.5;
    text-align: center;
    font-weight: 500;
  `;
  finalWarning.textContent = "If the hashes don't match exactly, or you're unsure about any detail, reject the transaction.";
  
  content.appendChild(finalWarning);
  
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
      return addr;
  }
  
  return address;
}
